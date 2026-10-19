package payment

import (
	"errors"
	"testing"

	"contgen/internal/telegram"
)

func TestRetryTelegramError(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "definitive client error", err: &telegram.APIError{Code: 400}, want: false},
		{name: "rate limit", err: &telegram.APIError{Code: 429}, want: true},
		{name: "server error", err: &telegram.APIError{Code: 500}, want: true},
		{name: "transport error", err: errors.New("network"), want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := retryTelegramError(test.err); got != test.want {
				t.Fatalf("retryTelegramError() = %v, want %v", got, test.want)
			}
		})
	}
}
