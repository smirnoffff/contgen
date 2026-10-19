package renderer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"contgen/internal/model"
	"contgen/internal/storage/dbgen"
)

const stderrLimit = 16 * 1024

// defaultRenderSeconds gives the racing-bar animation enough time to be
// readable in Telegram. It can be tuned for a local installation with
// RENDER_SECONDS (between 3 and 60 seconds) without changing the renderer
// protocol or the Python subprocess.
const defaultRenderSeconds = 12.0

type renderPoint struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Year  int     `json:"year"`
}

type renderRequest struct {
	Title      string        `json:"title"`
	Points     []renderPoint `json:"points"`
	Palette    Palette       `json:"palette"`
	Width      int           `json:"width"`
	Height     int           `json:"height"`
	FPS        int           `json:"fps"`
	Seconds    float64       `json:"seconds"`
	OutputPath string        `json:"output_path"`
}

// RenderRacingBar renders one video in a Python subprocess. Paths can be
// overridden with RENDERER_PYTHON and RENDERER_SCRIPT (see README.md).
func RenderRacingBar(ctx context.Context, datasetTitle string, points []model.Point, theme dbgen.RenderTheme, outputPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(points) == 0 {
		return fmt.Errorf("нет точек для рендеринга видео")
	}
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("не указан путь результата")
	}
	outputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("путь результата: %w", err)
	}

	// Keep the old output intact on failure; rename on the same filesystem.
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), ".contgen-render-*.mp4")
	if err != nil {
		return fmt.Errorf("временный файл рендера: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("закрытие временного файла: %w", err)
	}

	request := renderRequest{
		Title: datasetTitle, Palette: GetPalette(theme),
		Width: 1080, Height: 1920, FPS: 30, Seconds: configuredSeconds(),
		OutputPath: tmpPath, Points: make([]renderPoint, len(points)),
	}
	for i, p := range points {
		request.Points[i] = renderPoint{Name: p.EntityName, Value: p.Value, Year: p.RecordedAt}
	}
	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("данные рендера: %w", err)
	}

	python := envOr("RENDERER_PYTHON", ".venv/bin/python")
	script, err := filepath.Abs(envOr("RENDERER_SCRIPT", "python/renderer/main.py"))
	if err != nil {
		return fmt.Errorf("путь Python-скрипта: %w", err)
	}
	cmd := exec.CommandContext(ctx, python, "-u", script)
	cmd.Stdin = bytes.NewReader(data)
	diagnostics := &tailBuffer{}
	cmd.Stderr = diagnostics
	cmd.WaitDelay = 2 * time.Second
	configureProcessGroup(cmd)
	err = cmd.Run()
	// Also stop any child left behind by a failed or uncooperative script.
	stopProcessGroup(cmd)
	if ctx.Err() != nil {
		return fmt.Errorf("рендер отменён: %w", ctx.Err())
	}
	if err != nil {
		return fmt.Errorf("Python-рендерер (%s): %w\n%s", python, err, strings.TrimSpace(string(diagnostics.data)))
	}
	info, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("проверка результата рендера: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("Python-рендерер не создал видео")
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("сохранение видео: %w", err)
	}
	return nil
}

func configuredSeconds() float64 {
	raw := strings.TrimSpace(os.Getenv("RENDER_SECONDS"))
	if raw == "" {
		return defaultRenderSeconds
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || seconds < 3 || seconds > 60 {
		return defaultRenderSeconds
	}
	return seconds
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// tailBuffer keeps diagnostics bounded even when a child emits a large log.
type tailBuffer struct{ data []byte }

func (b *tailBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if n >= stderrLimit {
		b.data = append(b.data[:0], p[n-stderrLimit:]...)
		return n, nil
	}
	if overflow := len(b.data) + n - stderrLimit; overflow > 0 {
		b.data = append(b.data[:0], b.data[overflow:]...)
	}
	b.data = append(b.data, p...)
	return n, nil
}
