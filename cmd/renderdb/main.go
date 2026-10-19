package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"contgen/internal/config"
	"contgen/internal/model"
	"contgen/internal/renderer"
	"contgen/internal/storage"
	"contgen/internal/storage/dbgen"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Подключаемся к базе данных
	cfg := config.Load()
	db, err := storage.New(ctx, cfg.DSN())
	if err != nil {
		slog.Error("критическая ошибка запуска СУБД", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("✅ БД подключена")

	// 2. Выбираем системные датасеты (userID = nil)
	datasets, err := db.GetAvailableDatasets(ctx, nil)
	if err != nil || len(datasets) == 0 {
		slog.Error("❌ датасеты не найдены. Убедись, что база не пустая!", "err", err)
		os.Exit(1)
	}

	targetDataset := datasets[0]
	slog.Info("📁 Нашли датасет", "id", targetDataset.ID, "title", targetDataset.Title)

	// 3. Достаем все точки для этого датасета
	records, err := db.GetDataPointsByDatasetID(ctx, targetDataset.ID)
	if err != nil {
		slog.Error("❌ ошибка получения точек данных", "err", err)
		os.Exit(1)
	}
	points := make([]model.Point, len(records))
	for i, row := range records {
		points[i] = model.Point{EntityName: row.EntityName, Value: row.Value, RecordedAt: row.RecordedAt}
	}
	slog.Info("📊 Точки загружены", "кол-во", len(points))

	// 4. ЗАПУСКАЕМ СТАНОК РЕНДЕРА!
	slog.Info("🎬 Погнали рендерить видео...")
	startTime := time.Now()
	outputPath := "result.mp4"

	// Используем тему CyberHUD из сгенерированного enum.
	err = renderer.RenderRacingBar(ctx, targetDataset.Title, points, dbgen.RenderThemeCYBERHUD, outputPath)
	if err != nil {
		slog.Error("❌ ошибка рендеринга", "err", err)
		os.Exit(1)
	}

	slog.Info("🎉 ВИДЕО ГОТОВО!",
		"файл", outputPath,
		"время_выполнения", time.Since(startTime),
	)
}
