package discounts

import (
	"testing"
	"time"

	"github.com/dstryaaa/nhtk1/internal/models"
)

func TestLoyalty_Apply(t *testing.T) {
	// Фиксированные даты — никакого time.Now().
	createdAt := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	ptr := func(t time.Time) *time.Time { return &t }

	tests := []struct {
		name       string
		days       int
		percent    int64
		lastDate   *time.Time
		itemsTotal int64
		wantOK     bool
		wantAmount int64
		wantTotal  int64
	}{
		// Позитив.
		{"12 дней назад", 30, 5, ptr(createdAt.AddDate(0, 0, -12)), 1000, true, 50, 950},
		{"ровно 30 дней — граница включительна", 30, 5, ptr(createdAt.AddDate(0, 0, -30)), 1000, true, 50, 950},
		{"0 дней (сегодня)", 30, 5, ptr(createdAt), 1000, true, 50, 950},

		// Негатив.
		{"31 день — больше 30", 30, 5, ptr(createdAt.AddDate(0, 0, -31)), 1000, false, 0, 1000},
		{"lastPurchase = nil", 30, 5, nil, 1000, false, 0, 1000},
		{"будущая дата", 30, 5, ptr(createdAt.AddDate(0, 0, 5)), 1000, false, 0, 1000},
		{"ItemsTotal = 0", 30, 5, ptr(createdAt.AddDate(0, 0, -12)), 0, false, 0, 0},
		{"percent = 0", 30, 0, ptr(createdAt.AddDate(0, 0, -12)), 1000, false, 0, 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &models.DiscountContext{
				Order: &models.Order{
					Customer:  models.Customer{LastPurchaseDate: tt.lastDate},
					CreatedAt: createdAt,
				},
				ItemsTotal: tt.itemsTotal,
				Applied:    make([]models.AppliedDiscount, 0),
			}
			d := NewLoyalty("Лояльность", tt.days, tt.percent)

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
