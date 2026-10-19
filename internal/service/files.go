package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxAudioSize = 20 << 20

type AudioFile struct {
	Path        string
	DurationSec int
}

// SaveAudio stores a user upload below root and returns a relative path for
// music_tracks.file_path. The database never receives an absolute local path.
func SaveAudio(ctx context.Context, root, original string, src io.Reader) (AudioFile, error) {
	ext := strings.ToLower(filepath.Ext(original))
	if ext != ".mp3" && ext != ".wav" && ext != ".ogg" && ext != ".m4a" && ext != ".aac" {
		return AudioFile{}, fmt.Errorf("поддерживаются MP3, WAV, OGG, M4A и AAC")
	}
	if root == "" {
		root = "data"
	}
	dir := filepath.Join(root, "music")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return AudioFile{}, err
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return AudioFile{}, err
	}
	name := hex.EncodeToString(token[:]) + ext
	rel := filepath.ToSlash(filepath.Join("music", name))
	path := filepath.Join(root, filepath.FromSlash(rel))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return AudioFile{}, err
	}
	written, copyErr := io.CopyN(f, src, maxAudioSize+1)
	closeErr := f.Close()
	if copyErr != nil && copyErr != io.EOF {
		_ = os.Remove(path)
		return AudioFile{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return AudioFile{}, closeErr
	}
	if written > maxAudioSize {
		_ = os.Remove(path)
		return AudioFile{}, fmt.Errorf("аудиофайл больше 20 МБ")
	}
	duration, err := probeDuration(ctx, path)
	if err != nil {
		_ = os.Remove(path)
		return AudioFile{}, err
	}
	return AudioFile{Path: rel, DurationSec: duration}, nil
}

func probeDuration(ctx context.Context, path string) (int, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("не удалось прочитать длительность аудио")
	}
	var seconds float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &seconds); err != nil || seconds <= 0 || seconds > 600 {
		return 0, fmt.Errorf("длительность аудио должна быть от 1 секунды до 10 минут")
	}
	return int(seconds + 0.5), nil
}

func SafePath(root, relative string) (string, error) {
	if root == "" {
		root = "data"
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("путь выходит за пределы DATA_DIR")
	}
	return candidate, nil
}

func ValidateTitle(title string) error {
	if !utf8.ValidString(title) || len([]rune(title)) < 1 || len([]rune(title)) > 120 {
		return fmt.Errorf("название должно содержать от 1 до 120 символов")
	}
	return nil
}
