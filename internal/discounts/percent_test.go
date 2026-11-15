package discounts

import (
	"testing"

	"github.com/dstryaaa/nhtk1/internal/models"
)

// newCtx создаёт контекст с заданной суммой товаров — для тестов стратегий.
func newCtx(itemsTotal int64) *models.DiscountContext {
	return &models.DiscountContext{
		ItemsTotal: itemsTotal,
		Applied:    make([]models.AppliedDiscount, 0),
	}
}

func TestPercentDiscount_Apply(t *testing.T) {
	tests := []struct {
		name       string
		percent    int64
		itemsTotal int64
		wantOK     bool
		wantAmount int64
		wantTotal  int64
	}{
		// Позитивные сценарии.
		{"10% от 1000", 10, 1000, true, 100, 900},
		{"100% от 1000", 100, 1000, true, 1000, 0},
		{"5% от 10000", 5, 10000, true, 500, 9500},

		// Негативные: скидка не применяется.
		{"10% от 0", 10, 0, false, 0, 0},
		{"0% от 1000", 0, 1000, false, 0, 1000},
		{"отрицательный процент", -5, 1000, false, 0, 1000},

		// Округление: amount = 0 после округления → не применяем.
		// 1 * 90 / 100 = 0.9 → roundHalfToEven(90,100) = 1, amount = 0.
		{"10% от 1 (amount=0)", 10, 1, false, 0, 1},

		// Half-to-even: 1 * 50 / 100 = 0.5 → 0 (чётное), amount = 1.
		{"50% от 1 (half-to-even)", 50, 1, true, 1, 0},

		// Баланс old - amount == new.
		{"33% от 100", 33, 100, true, 33, 67},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newCtx(tt.itemsTotal)
			d := NewPercentDiscount("тест", tt.percent)

			ad, ok := d.Apply(ctx)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				// При отказе контекст не должен меняться.
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
			// Инвариант: old - amount == new.
			if tt.itemsTotal-ad.Amount != ctx.ItemsTotal {
				t.Errorf("баланс не сходится: %d - %d != %d", tt.itemsTotal, ad.Amount, ctx.ItemsTotal)
			}
			if ad.Name != "тест" {
				t.Errorf("Name = %q, want %q", ad.Name, "тест")
			}
			if ad.Reason == "" {
				t.Error("Reason пустой")
			}
		})
	}
}
