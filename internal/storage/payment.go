package storage

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"contgen/internal/storage/dbgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrPromoUnavailable = errors.New("промокод недействителен, исчерпан или уже использован")
	ErrPaymentPending   = errors.New("для корзины уже создан платёж")
	ErrPaymentInvalid   = errors.New("платёж не найден или уже обработан")
)

// PreparePayment validates the cart, calculates its total from database rows,
// reserves a valid promo usage and creates the payment row. The returned
// external ID is a stable Telegram invoice payload; it is never taken from a
// browser form.
func (s *Storage) PreparePayment(ctx context.Context, userID, orderID int64, promoCode string) (dbgen.Payment, error) {
	return s.PreparePaymentWithMethod(ctx, userID, orderID, dbgen.PaymentMethodSTARS, promoCode)
}

// PreparePaymentWithMethod prepares a payment for one of the user-facing
// payment methods. Promo codes can still turn the final payment into PROMO
// when they cover the complete order.
func (s *Storage) PreparePaymentWithMethod(ctx context.Context, userID, orderID int64, method dbgen.PaymentMethod, promoCode string) (dbgen.Payment, error) {
	var zero dbgen.Payment
	if userID <= 0 || orderID <= 0 {
		return zero, ErrInvalid
	}
	if method != dbgen.PaymentMethodSTARS && method != dbgen.PaymentMethodADS {
		return zero, ErrInvalid
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback(ctx)
	q := s.WithTx(tx)
	if _, err := q.LockUser(ctx, userID); err != nil {
		return zero, unavailable(err)
	}
	order, err := q.LockOwnedOrder(ctx, dbgen.LockOwnedOrderParams{ID: orderID, UserID: userID})
	if err != nil {
		return zero, unavailable(err)
	}
	if order.Status != dbgen.OrderStatusCART {
		return zero, ErrPaymentInvalid
	}
	if pending, err := q.GetPendingPaymentForOrder(ctx, orderID); err == nil {
		return pending, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return zero, err
	}
	count, err := q.CountOrderItems(ctx, orderID)
	if err != nil {
		return zero, err
	}
	if count == 0 {
		return zero, ErrEmptyCart
	}
	if count > MaxCartItems {
		return zero, ErrCartFull
	}
	invalid, err := q.CountUnusableCartItems(ctx, dbgen.CountUnusableCartItemsParams{OrderID: orderID, UserID: &userID})
	if err != nil {
		return zero, err
	}
	if invalid > 0 {
		return zero, ErrUnavailableResource
	}

	grossCents := count * RenderPrice * 100
	var promoID *int64
	var discountPercent *int
	var discountCents int64
	code := strings.ToUpper(strings.TrimSpace(promoCode))
	if code != "" {
		promo, err := q.LockValidPromocode(ctx, code)
		if errors.Is(err, pgx.ErrNoRows) {
			return zero, ErrPromoUnavailable
		}
		if err != nil {
			return zero, err
		}
		used, err := q.HasUsedPromocode(ctx, dbgen.HasUsedPromocodeParams{PromocodeID: promo.ID, UserID: userID})
		if err != nil {
			return zero, err
		}
		if used {
			return zero, ErrPromoUnavailable
		}
		promoID, discountPercent = &promo.ID, &promo.DiscountPercent
		discountCents = grossCents * int64(promo.DiscountPercent) / 100
		if err := q.CreatePromocodeUsage(ctx, dbgen.CreatePromocodeUsageParams{PromocodeID: promo.ID, UserID: userID, OrderID: orderID}); err != nil {
			return zero, ErrPromoUnavailable
		}
		if consumed, err := q.ConsumePromocode(ctx, promo.ID); err != nil {
			return zero, err
		} else if consumed != 1 {
			return zero, ErrPromoUnavailable
		}
	}
	total := numericCents(grossCents - discountCents)
	discount := numericCents(discountCents)
	if updated, err := q.SetOrderCheckoutTotals(ctx, dbgen.SetOrderCheckoutTotalsParams{ID: orderID, PromocodeID: promoID, DiscountPercentApplied: discountPercent, DiscountAmount: discount, TotalAmount: total, UserID: userID}); err != nil {
		return zero, err
	} else if updated != 1 {
		return zero, ErrPaymentInvalid
	}
	if grossCents-discountCents == 0 {
		method = dbgen.PaymentMethodPROMO
	}
	payload := fmt.Sprintf("contgen:%d:%d:%d", userID, orderID, time.Now().UnixNano())
	payment, err := q.CreatePayment(ctx, dbgen.CreatePaymentParams{OrderID: orderID, Method: method, ExternalID: &payload, Amount: total})
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, err
	}
	return payment, nil
}

// CompletePayment is called by the Telegram update processor after a
// successful payment. It is idempotent: a repeated Telegram update leaves the
// already completed order untouched.
func (s *Storage) CompletePayment(ctx context.Context, payload, chargeID string) error {
	if strings.TrimSpace(payload) == "" || strings.TrimSpace(chargeID) == "" {
		return ErrPaymentInvalid
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.WithTx(tx)
	paymentPayload := payload
	payment, err := q.GetPaymentByExternalID(ctx, &paymentPayload)
	if errors.Is(err, pgx.ErrNoRows) {
		charge := chargeID
		payment, err = q.GetPaymentByChargeID(ctx, &charge)
		if err == nil && payment.Status == dbgen.PaymentStatusSUCCEEDED {
			return tx.Commit(ctx)
		}
		return ErrPaymentInvalid
	}
	if err != nil {
		return err
	}
	if payment.Status == dbgen.PaymentStatusSUCCEEDED {
		return tx.Commit(ctx)
	}
	if payment.Status != dbgen.PaymentStatusPENDING {
		return ErrPaymentInvalid
	}
	order, err := q.LockOrderByID(ctx, payment.OrderID)
	if err != nil {
		return unavailable(err)
	}
	if order.Status == dbgen.OrderStatusPAID {
		return tx.Commit(ctx)
	}
	if order.Status != dbgen.OrderStatusCART {
		return ErrPaymentInvalid
	}
	if n, err := q.MarkPaymentSucceeded(ctx, dbgen.MarkPaymentSucceededParams{ID: payment.ID, ExternalID: &chargeID}); err != nil {
		return err
	} else if n != 1 {
		return ErrPaymentInvalid
	}
	if n, err := q.SetOrderPaidByPayment(ctx, dbgen.SetOrderPaidByPaymentParams{ID: order.ID, PaymentMethod: &payment.Method}); err != nil {
		return err
	} else if n != 1 {
		return ErrPaymentInvalid
	}
	return tx.Commit(ctx)
}

func numericCents(cents int64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(cents), Exp: -2, Valid: true}
}

// CheckPayment verifies the exact pending Stars invoice. The error is kept
// separate from the boolean result so the Telegram poller can retry a
// temporary database outage instead of acknowledging an update it could not
// persist.
func (s *Storage) CheckPayment(ctx context.Context, payload string, stars int64) (bool, error) {
	if payload == "" || stars <= 0 {
		return false, nil
	}
	p := payload
	payment, err := s.GetPaymentByExternalID(ctx, &p)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if payment.Status != dbgen.PaymentStatusPENDING || payment.Method != dbgen.PaymentMethodSTARS {
		return false, nil
	}
	value, err := payment.Amount.Float64Value()
	if err != nil {
		return false, err
	}
	return value.Valid && value.Float64 == float64(stars), nil
}

// ValidatePayment checks the exact pending invoice before Telegram's ten-second
// pre-checkout deadline.
func (s *Storage) ValidatePayment(ctx context.Context, payload string, stars int64) bool {
	ok, err := s.CheckPayment(ctx, payload, stars)
	return err == nil && ok
}

func (s *Storage) FailPayment(ctx context.Context, paymentID int64) error {
	if paymentID <= 0 {
		return ErrPaymentInvalid
	}
	n, err := s.MarkPaymentFailed(ctx, paymentID)
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrPaymentInvalid
	}
	return nil
}

// CancelPayment releases a promo reservation when an invoice could not be
// created. A user who simply leaves Telegram can reuse the same pending invoice.
func (s *Storage) CancelPayment(ctx context.Context, paymentID int64) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.WithTx(tx)
	payment, err := q.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return unavailable(err)
	}
	if payment.Status != dbgen.PaymentStatusPENDING {
		return nil
	}
	order, err := q.LockOrderByID(ctx, payment.OrderID)
	if err != nil {
		return unavailable(err)
	}
	if order.PromocodeID != nil {
		if _, err := q.DeletePromocodeUsage(ctx, dbgen.DeletePromocodeUsageParams{PromocodeID: *order.PromocodeID, OrderID: order.ID}); err != nil {
			return err
		}
		if _, err := q.ReleasePromocode(ctx, *order.PromocodeID); err != nil {
			return err
		}
	}
	if _, err := q.ResetOrderCheckoutTotals(ctx, order.ID); err != nil {
		return err
	}
	if _, err := q.MarkPaymentFailed(ctx, payment.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
