package discounts

import (
	"testing"
	"time"

	"github.com/dstryaaa/nhtk1/internal/models"
)

func TestPromoCode_Apply(t *testing.T) {
	validUntil := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		percent      int64
		promoValid   bool
		promoApplied bool
		itemsTotal   int64
		wantOK       bool
		wantAmount   int64
		wantTotal    int64
		wantApplied  bool
	}{
		// Позитив: валиден, не применён — применяется.
		{"валидный 15% от 1000", 15, true, false, 1000, true, 150, 850, false},

		// Негатив: невалиден.
		{"невалидный", 15, false, false, 1000, false, 0, 1000, false},
		// Негатив: уже применён.
		{"уже применён", 15, true, true, 1000, false, 0, 1000, true},
		// Негатив: ItemsTotal 0.
		{"ItemsTotal 0", 15, true, false, 0, false, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &models.DiscountContext{
				ItemsTotal:   tt.itemsTotal,
				Applied:      make([]models.AppliedDiscount, 0),
				PromoValid:   tt.promoValid,
				PromoApplied: tt.promoApplied,
			}
			inner := NewPercentDiscount("внутренняя", tt.percent)
			d := NewPromoCode("Промокод", "SAVE15", &validUntil, true, inner)

			ad, ok := d.Apply(ctx)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				if ctx.ItemsTotal != tt.itemsTotal {
					t.Errorf("ItemsTotal изменился при отказе")
				}
				// Стратегия не должна ставить PromoApplied.
				if ctx.PromoApplied != tt.wantApplied {
					t.Errorf("PromoApplied = %v, want %v (стратегия не должна его менять)", ctx.PromoApplied, tt.wantApplied)
				}
				return
			}
			if ad.Amount != tt.wantAmount {
				t.Errorf("Amount = %d, want %d", ad.Amount, tt.wantAmount)
			}
			if ctx.ItemsTotal != tt.wantTotal {
				t.Errorf("ItemsTotal = %d, want %d", ctx.ItemsTotal, tt.wantTotal)
			}
			if ad.Name != "Промокод" {
				t.Errorf("Name = %q, want %q", ad.Name, "Промокод")
			}
			// Стратегия не должна ставить PromoApplied — это делает движок.
			if ctx.PromoApplied != false {
				t.Errorf("PromoCode.Apply не должен ставить PromoApplied")
			}
		})
	}
}
