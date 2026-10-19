package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var tokenPattern = regexp.MustCompile(`^[0-9]+:[A-Za-z0-9_-]+$`)

// Client sends only explicitly requested messages. It never broadcasts or
// automatically sends a finished render; callers supply the authenticated user.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

type Bot struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

type APIError struct {
	Code        int
	Description string
	RetryAfter  int
}

func (e *APIError) Error() string { return fmt.Sprintf("Telegram (%d): %s", e.Code, e.Description) }

func NewClient(token string) (*Client, error) {
	if !tokenPattern.MatchString(token) {
		return nil, fmt.Errorf("некорректный токен Telegram-бота")
	}
	return &Client{token: token, baseURL: "https://api.telegram.org", http: &http.Client{Timeout: 2 * time.Minute}}, nil
}

func (c *Client) GetMe(ctx context.Context) (Bot, error) {
	var bot Bot
	err := c.call(ctx, "getMe", "", nil, &bot)
	if err == nil && (!bot.IsBot || bot.ID <= 0) {
		return Bot{}, fmt.Errorf("некорректный ответ Telegram")
	}
	return bot, err
}

// SendVideo uploads a local MP4 into this user's private chat. The positive ID
// restriction prevents a route from being repurposed for groups or channels.
// filePath must be obtained after ownership checks.
func (c *Client) SendVideo(ctx context.Context, chatID int64, filePath, caption string) (string, error) {
	if chatID <= 0 {
		return "", fmt.Errorf("отправка доступна после входа через Telegram")
	}
	if utf8.RuneCountInString(caption) > 1024 {
		return "", fmt.Errorf("слишком длинная подпись видео")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("видео недоступно: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 50<<20 {
		return "", fmt.Errorf("видео должно быть обычным файлом размером до 50 МБ")
	}
	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	contentType := multipartWriter.FormDataContentType()
	done := make(chan error, 1)
	go func() {
		err := writeVideo(multipartWriter, f, chatID, caption)
		if err == nil {
			err = multipartWriter.Close()
		}
		_ = writer.CloseWithError(err)
		done <- err
	}()
	var message struct {
		Video struct {
			FileID string `json:"file_id"`
		} `json:"video"`
	}
	err = c.call(ctx, "sendVideo", contentType, reader, &message)
	// Unblock an upload if Telegram rejects it before consuming the whole body.
	_ = reader.Close()
	writeErr := <-done
	if err != nil {
		return "", err
	}
	if writeErr != nil {
		return "", fmt.Errorf("загрузка видео прервана")
	}
	if message.Video.FileID == "" {
		return "", fmt.Errorf("Telegram не вернул идентификатор видео")
	}
	return message.Video.FileID, nil
}

func writeVideo(w *multipart.Writer, f *os.File, chatID int64, caption string) error {
	fields := [][2]string{{"chat_id", strconv.FormatInt(chatID, 10)}}
	if caption != "" {
		// Telegram renders a supplied caption below the video. Keep the field
		// out of the multipart request when callers want a clean video-only
		// message; an empty caption still creates an unnecessary caption field.
		fields = append(fields, [2]string{"caption", caption})
	}
	fields = append(fields, [2]string{"supports_streaming", "true"})
	for _, field := range fields {
		if err := w.WriteField(field[0], field[1]); err != nil {
			return err
		}
	}
	part, err := w.CreateFormFile("video", "contgen.mp4")
	if err != nil {
		return err
	}
	_, err = io.Copy(part, f)
	return err
}

func (c *Client) call(ctx context.Context, method, contentType string, body io.Reader, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/bot"+c.token+"/"+method, body)
	if err != nil {
		return fmt.Errorf("не удалось подготовить запрос Telegram")
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	response, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// net/http's URL errors contain the bot token. Never log or return them.
		return fmt.Errorf("Telegram недоступен, повторите отправку позже")
	}
	defer response.Body.Close()
	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Code        int             `json:"error_code"`
		Description string          `json:"description"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 || json.Unmarshal(data, &envelope) != nil {
		return fmt.Errorf("некорректный ответ Telegram")
	}
	if !envelope.OK || response.StatusCode < 200 || response.StatusCode >= 300 {
		if envelope.Code == 0 {
			envelope.Code = response.StatusCode
		}
		description := strings.ReplaceAll(envelope.Description, c.token, "[token]")
		if len(description) > 500 {
			description = description[:500]
		}
		return &APIError{Code: envelope.Code, Description: description, RetryAfter: envelope.Parameters.RetryAfter}
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return errors.New("пустой ответ Telegram")
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("некорректный результат Telegram")
	}
	return nil
}
