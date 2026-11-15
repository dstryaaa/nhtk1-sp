package discounts

import (
	"testing"

	"github.com/dstryaaa/nhtk1/internal/models"
)

func TestFreeDelivery_Apply(t *testing.T) {
	tests := []struct {
		name         string
		threshold    int64
		itemsTotal   int64
		deliveryCost int64
		wantOK       bool
		wantAmount   int64
		wantDelivery int64
	}{
		// Позитив: строго больше порога, доставка обнуляется.
		{"5001 > 5000", 5000, 5001, 500, true, 500, 0},
		{"10000 > 5000", 5000, 10000, 300, true, 300, 0},

		// Граница: строго больше, ровно порог — не применяется.
		{"ровно 5000", 5000, 5000, 500, false, 0, 500},

		// Негатив.
		{"4999 < 5000", 5000, 4999, 500, false, 0, 500},
		{"доставка уже 0", 5000, 10000, 0, false, 0, 0},
		{"пустая корзина", 5000, 0, 500, false, 0, 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &models.DiscountContext{
				ItemsTotal:   tt.itemsTotal,
				DeliveryCost: tt.deliveryCost,
				Applied:      make([]models.AppliedDiscount, 0),
			}
			d := NewFreeDelivery(tt.threshold, "тест")

			ad, ok := d.Apply(ctx)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				if ctx.DeliveryCost != tt.deliveryCost {
					t.Errorf("DeliveryCost изменился при отказе")
				}
				return
			}
			if ad.Amount != tt.wantAmount {
				t.Errorf("Amount = %d, want %d", ad.Amount, tt.wantAmount)
			}
			if ctx.DeliveryCost != tt.wantDelivery {
				t.Errorf("DeliveryCost = %d, want %d", ctx.DeliveryCost, tt.wantDelivery)
			}
			// ItemsTotal не должен меняться.
			if ctx.ItemsTotal != tt.itemsTotal {
				t.Errorf("ItemsTotal изменился: %d, want %d", ctx.ItemsTotal, tt.itemsTotal)
			}
		})
	}
}
