package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

type InvoicePrice struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

// CreateInvoiceLink creates a Telegram Stars invoice. Currency XTR is fixed
// by Telegram and provider_token is intentionally omitted for Stars.
func (c *Client) CreateInvoiceLink(ctx context.Context, title, description, payload string, amount int64) (string, error) {
	if amount <= 0 || amount > 10000000 {
		return "", fmt.Errorf("некорректная сумма счёта")
	}
	body, err := json.Marshal(struct {
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Payload     string         `json:"payload"`
		Currency    string         `json:"currency"`
		Prices      []InvoicePrice `json:"prices"`
	}{Title: title, Description: description, Payload: payload, Currency: "XTR", Prices: []InvoicePrice{{Label: "CONTGEN", Amount: amount}}})
	if err != nil {
		return "", err
	}
	var link string
	if err := c.call(ctx, "createInvoiceLink", "application/json", bytes.NewReader(body), &link); err != nil {
		return "", err
	}
	if link == "" {
		return "", fmt.Errorf("Telegram не вернул ссылку на оплату")
	}
	return link, nil
}

type Update struct {
	ID               int64             `json:"update_id"`
	PreCheckoutQuery *PreCheckoutQuery `json:"pre_checkout_query"`
	Message          *Message          `json:"message"`
}

type PreCheckoutQuery struct {
	ID             string `json:"id"`
	From           User   `json:"from"`
	Currency       string `json:"currency"`
	TotalAmount    int64  `json:"total_amount"`
	InvoicePayload string `json:"invoice_payload"`
}

type Message struct {
	Chat              Chat               `json:"chat"`
	SuccessfulPayment *SuccessfulPayment `json:"successful_payment"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type SuccessfulPayment struct {
	Currency                string `json:"currency"`
	TotalAmount             int64  `json:"total_amount"`
	InvoicePayload          string `json:"invoice_payload"`
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
}

func (c *Client) GetUpdates(ctx context.Context, offset int64) ([]Update, error) {
	body, err := json.Marshal(struct {
		Offset         int64    `json:"offset"`
		Timeout        int      `json:"timeout"`
		AllowedUpdates []string `json:"allowed_updates"`
	}{Offset: offset, Timeout: 30, AllowedUpdates: []string{"pre_checkout_query", "message"}})
	if err != nil {
		return nil, err
	}
	var updates []Update
	if err := c.call(ctx, "getUpdates", "application/json", bytes.NewReader(body), &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func (c *Client) AnswerPreCheckoutQuery(ctx context.Context, id string, ok bool, message string) error {
	body, err := json.Marshal(struct {
		ID           string `json:"pre_checkout_query_id"`
		OK           bool   `json:"ok"`
		ErrorMessage string `json:"error_message,omitempty"`
	}{ID: id, OK: ok, ErrorMessage: message})
	if err != nil {
		return err
	}
	var result bool
	return c.call(ctx, "answerPreCheckoutQuery", "application/json", bytes.NewReader(body), &result)
}
