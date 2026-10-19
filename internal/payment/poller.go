// Package payment connects Telegram's update stream to the storage state
// machine. Telegram confirms the charge; storage atomically marks the payment,
// order and already-created render jobs as paid.
package payment

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"contgen/internal/storage"
	"contgen/internal/telegram"
)

func Run(ctx context.Context, bot *telegram.Client, store *storage.Storage) error {
	if bot == nil || store == nil {
		return errors.New("для обработки платежей нужны Telegram и PostgreSQL")
	}
	var offset int64
	for ctx.Err() == nil {
		pollCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
		updates, err := bot.GetUpdates(pollCtx, offset)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("Telegram updates", "err", err)
			if !wait(ctx, 3*time.Second) {
				return nil
			}
			continue
		}
		for _, update := range updates {
			if update.ID < offset {
				continue
			}
			retry := false
			if query := update.PreCheckoutQuery; query != nil {
				checkCtx, checkCancel := context.WithTimeout(ctx, 5*time.Second)
				valid := store.ValidatePayment(checkCtx, query.InvoicePayload, query.TotalAmount)
				checkCancel()
				ok := query.Currency == "XTR" && strings.HasPrefix(query.InvoicePayload, "contgen:") && valid
				message := "Счёт устарел. Создай его заново."
				if ok {
					message = ""
				}
				answerCtx, answerCancel := context.WithTimeout(ctx, 8*time.Second)
				err := bot.AnswerPreCheckoutQuery(answerCtx, query.ID, ok, message)
				answerCancel()
				if err != nil {
					slog.Error("подтверждение Telegram платежа", "err", err)
					// An expired/invalid query must not block later updates forever.
					// Retry transport and rate-limit failures, but consume Telegram's
					// definitive 4xx response.
					retry = retryTelegramError(err)
				}
			}
			if !retry && update.Message != nil && update.Message.SuccessfulPayment != nil {
				payment := update.Message.SuccessfulPayment
				if payment.Currency != "XTR" || payment.TotalAmount <= 0 || payment.InvoicePayload == "" || payment.TelegramPaymentChargeID == "" {
					slog.Warn("пропущено некорректное Telegram подтверждение платежа", "update_id", update.ID)
				} else {
					checkCtx, checkCancel := context.WithTimeout(ctx, 5*time.Second)
					valid, err := store.CheckPayment(checkCtx, payment.InvoicePayload, payment.TotalAmount)
					checkCancel()
					if err != nil {
						slog.Error("проверка Telegram платежа", "err", err)
						retry = true
					} else if !valid {
						slog.Warn("пропущено несоответствующее Telegram подтверждение платежа", "update_id", update.ID)
					} else {
						completeCtx, completeCancel := context.WithTimeout(ctx, 10*time.Second)
						err = store.CompletePayment(completeCtx, payment.InvoicePayload, payment.TelegramPaymentChargeID)
						completeCancel()
						if err != nil {
							if errors.Is(err, storage.ErrPaymentInvalid) {
								slog.Warn("Telegram платеж уже обработан или недействителен", "update_id", update.ID)
							} else {
								slog.Error("фиксация Telegram платежа", "err", err)
								retry = true
							}
						} else {
							slog.Info("Telegram платеж подтверждён", "payload", payment.InvoicePayload)
						}
					}
				}
			}
			if retry {
				break
			}
			offset = update.ID + 1
		}
	}
	return nil
}

func retryTelegramError(err error) bool {
	var apiErr *telegram.APIError
	if errors.As(err, &apiErr) && apiErr.Code >= 400 && apiErr.Code < 500 && apiErr.Code != 429 {
		return false
	}
	return true
}

func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
