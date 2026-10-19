package storage

import (
	"context"
	"fmt"

	"contgen/internal/storage/dbgen"
)

func (s *Storage) MarkJobCompleted(ctx context.Context, jobID int64, tgFileID string) error {
	affected, err := s.SetJobCompleted(ctx, dbgen.SetJobCompletedParams{ID: jobID, TelegramFileID: &tgFileID})
	if err != nil {
		return fmt.Errorf("ошибка перевода задачи #%d в COMPLETED: %w", jobID, err)
	}
	if affected == 0 {
		return fmt.Errorf("задача #%d не найдена для обновления", jobID)
	}
	return nil
}
