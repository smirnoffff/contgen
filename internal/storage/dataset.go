package storage

import (
	"context"
	"fmt"

	"contgen/internal/model"
	"contgen/internal/storage/dbgen"
)

// CreateDatasetWithPoints saves the dataset and its points atomically.
func (s *Storage) CreateDatasetWithPoints(ctx context.Context, userID *int64, title string, points []model.Point) (*dbgen.Dataset, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка старта транзакции: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.WithTx(tx)
	dataset, err := qtx.CreateDataset(ctx, dbgen.CreateDatasetParams{UserID: userID, Title: title})
	if err != nil {
		return nil, fmt.Errorf("ошибка вставки заголовка датасета: %w", err)
	}
	rows := make([]dbgen.CopyDataPointsParams, len(points))
	for i, p := range points {
		rows[i] = dbgen.CopyDataPointsParams{DatasetID: dataset.ID, EntityName: p.EntityName, Value: p.Value, RecordedAt: p.RecordedAt}
	}
	if _, err := qtx.CopyDataPoints(ctx, rows); err != nil {
		return nil, fmt.Errorf("ошибка пакетной вставки точек данных: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("ошибка фиксации транзакции: %w", err)
	}
	return &dataset, nil
}
