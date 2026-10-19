package storage

import (
	"context"
	"fmt"

	"contgen/internal/storage/dbgen"
)

func (s *Storage) ApplyPromocode(ctx context.Context, orderID, userID, promoID int64, discountPercent int) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка старта транзакции промокода: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.WithTx(tx)
	if err := qtx.SetOrderPromocode(ctx, dbgen.SetOrderPromocodeParams{ID: orderID, PromocodeID: &promoID, DiscountPercentApplied: &discountPercent}); err != nil {
		return fmt.Errorf("не удалось обновить заказ: %w", err)
	}
	if err := qtx.IncrementPromocodeUsage(ctx, promoID); err != nil {
		return fmt.Errorf("не удалось обновить счетчик промокода: %w", err)
	}
	if err := qtx.CreatePromocodeUsage(ctx, dbgen.CreatePromocodeUsageParams{PromocodeID: promoID, UserID: userID, OrderID: orderID}); err != nil {
		return fmt.Errorf("вы уже использовали этот промокод: %w", err)
	}
	return tx.Commit(ctx)
}
