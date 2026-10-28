package web

import (
	"contgen/internal/storage/dbgen"
	"testing"
	"time"
)

func TestPromoStateMatchesRedeemability(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	limit := 2
	tests := []struct {
		name  string
		promo dbgen.Promocode
		want  string
	}{
		{"active", dbgen.Promocode{IsActive: true}, "active"},
		{"disabled", dbgen.Promocode{IsActive: false}, "disabled"},
		{"expired", dbgen.Promocode{IsActive: true, ValidUntil: &past}, "expired"},
		{"expiry boundary", dbgen.Promocode{IsActive: true, ValidUntil: &now}, "expired"},
		{"exhausted", dbgen.Promocode{IsActive: true, MaxUses: &limit, UsedCount: 2}, "exhausted"},
		{"remaining", dbgen.Promocode{IsActive: true, MaxUses: &limit, UsedCount: 1, ValidUntil: &future}, "active"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := promoState(tt.promo, now); got != tt.want {
				t.Fatalf("state=%s, want %s", got, tt.want)
			}
		})
	}
}
