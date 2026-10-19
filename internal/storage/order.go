package storage

import (
	"context"
	"fmt"

	"contgen/internal/storage/dbgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Storage) GetOrCreateCart(ctx context.Context, userID int64) (*dbgen.Order, error) {
	order, err := s.EnsureCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания корзины: %w", err)
	}
	return &order, nil
}

func (s *Storage) MarkOrderPaid(ctx context.Context, orderID int64, method dbgen.PaymentMethod, totalAmount pgtype.Numeric) error {
	affected, err := s.SetOrderPaid(ctx, dbgen.SetOrderPaidParams{ID: orderID, PaymentMethod: &method, TotalAmount: totalAmount})
	if err != nil {
		return fmt.Errorf("ошибка оплаты заказа #%d: %w", orderID, err)
	}
	if affected == 0 {
		return fmt.Errorf("заказ #%d не найден или уже был оплачен", orderID)
	}
	return nil
}
