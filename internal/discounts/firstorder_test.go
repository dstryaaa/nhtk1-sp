package discounts

import (
	"testing"

	"github.com/dstryaaa/nhtk1/internal/models"
)

func TestFirstOrder_Apply(t *testing.T) {
	tests := []struct {
		name         string
		percent      int64
		isFirstOrder bool
		itemsTotal   int64
		wantOK       bool
		wantAmount   int64
		wantTotal    int64
	}{
		{"первый заказ 10% от 1000", 10, true, 1000, true, 100, 900},
		{"не первый заказ", 10, false, 1000, false, 0, 1000},
		{"первый заказ, ItemsTotal 0", 10, true, 0, false, 0, 0},
		{"первый заказ, percent 0", 0, true, 1000, false, 0, 1000},
		{"первый заказ 100%", 100, true, 1000, true, 1000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &models.DiscountContext{
				Order: &models.Order{
					Customer: models.Customer{IsFirstOrder: tt.isFirstOrder},
				},
				ItemsTotal: tt.itemsTotal,
				Applied:    make([]models.AppliedDiscount, 0),
			}
			d := NewFirstOrder("Первый заказ", tt.percent)

			ad, ok := d.Apply(ctx)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				if ctx.ItemsTotal != tt.itemsTotal {
					t.Errorf("ItemsTotal изменился при отказе")
				}
				return
			}
			if ad.Amount != tt.wantAmount {
				t.Errorf("Amount = %d, want %d", ad.Amount, tt.wantAmount)
			}
			if ctx.ItemsTotal != tt.wantTotal {
				t.Errorf("ItemsTotal = %d, want %d", ctx.ItemsTotal, tt.wantTotal)
			}
		})
	}
}
