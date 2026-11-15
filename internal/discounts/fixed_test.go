package discounts

import (
	"testing"
)

func TestFixedDiscount_Apply(t *testing.T) {
	tests := []struct {
		name       string
		value      int64
		threshold  int64
		itemsTotal int64
		wantOK     bool
		wantAmount int64
		wantTotal  int64
	}{
		// Позитив: порог достигнут, скидка применяется.
		{"500 от 3000", 500, 3000, 3000, true, 500, 2500},
		{"500 от 5000", 500, 3000, 5000, true, 500, 4500},
		{"порог 0", 500, 0, 1000, true, 500, 500},

		// Негатив: порог не достигнут.
		{"500 от 2999", 500, 3000, 2999, false, 0, 2999},
		{"500 от 0", 500, 3000, 0, false, 0, 0},
		{"value 0", 0, 1000, 1000, false, 0, 1000},
		{"value отрицательный", -500, 1000, 1000, false, 0, 1000},

		// Нижняя граница: скидка урезается до ItemsTotal.
		{"5000 от 1000 (урезание)", 5000, 1000, 1000, true, 1000, 0},
		{"5000 от 2000 (урезание)", 5000, 1000, 2000, true, 2000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newCtx(tt.itemsTotal)
			d := NewFixedDiscount(tt.value, tt.threshold, "тест")

			ad, ok := d.Apply(ctx)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				if ctx.ItemsTotal != tt.itemsTotal {
					t.Errorf("ItemsTotal изменился при отказе: %d, want %d", ctx.ItemsTotal, tt.itemsTotal)
				}
				return
			}
			if ad.Amount != tt.wantAmount {
				t.Errorf("Amount = %d, want %d", ad.Amount, tt.wantAmount)
			}
			if ctx.ItemsTotal != tt.wantTotal {
				t.Errorf("ItemsTotal = %d, want %d", ctx.ItemsTotal, tt.wantTotal)
			}
			if tt.itemsTotal-ad.Amount != ctx.ItemsTotal {
				t.Errorf("баланс не сходится")
			}
		})
	}
}
