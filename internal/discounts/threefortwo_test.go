package discounts

import (
	"testing"

	"github.com/dstryaaa/nhtk1/internal/models"
)

func cartItem(price int64, qty int, category string) models.CartItem {
	return models.CartItem{
		Product:  &models.Product{BasePrice: price, Category: category},
		Quantity: qty,
	}
}

func ctxWithItems3(items []models.CartItem, itemsTotal int64) *models.DiscountContext {
	return &models.DiscountContext{
		Order:      &models.Order{Items: items},
		ItemsTotal: itemsTotal,
		Applied:    make([]models.AppliedDiscount, 0),
	}
}

func TestThreeForTwo_Apply(t *testing.T) {
	tests := []struct {
		name       string
		category   string
		items      []models.CartItem
		itemsTotal int64
		wantOK     bool
		wantAmount int64
		wantTotal  int64
	}{
		{
			name:       "5 книг разных цен — 1 бесплатная (самая дешёвая)",
			category:   "Книги",
			items:      []models.CartItem{cartItem(100, 1, "Книги"), cartItem(200, 1, "Книги"), cartItem(300, 1, "Книги"), cartItem(400, 1, "Книги"), cartItem(500, 1, "Книги")},
			itemsTotal: 1500, wantOK: true, wantAmount: 100, wantTotal: 1400,
		},
		{
			name:       "6 книг — 2 бесплатные",
			category:   "Книги",
			items:      []models.CartItem{cartItem(100, 1, "Книги"), cartItem(200, 1, "Книги"), cartItem(300, 1, "Книги"), cartItem(400, 1, "Книги"), cartItem(500, 1, "Книги"), cartItem(150, 1, "Книги")},
			itemsTotal: 1650, wantOK: true, wantAmount: 250, wantTotal: 1400, // 100 + 150
		},
		{
			name:       "2 книги — не применяется",
			category:   "Книги",
			items:      []models.CartItem{cartItem(100, 1, "Книги"), cartItem(200, 1, "Книги")},
			itemsTotal: 300, wantOK: false, wantAmount: 0, wantTotal: 300,
		},
		{
			name:       "ловушка: 3×100 + 1×50 — бесплатная 50, не 100",
			category:   "Книги",
			items:      []models.CartItem{cartItem(100, 3, "Книги"), cartItem(50, 1, "Книги")},
			itemsTotal: 350, wantOK: true, wantAmount: 50, wantTotal: 300,
		},
		{
			name:       "чужая категория не участвует",
			category:   "Книги",
			items:      []models.CartItem{cartItem(100, 3, "Книги"), cartItem(500, 2, "Футболки")},
			itemsTotal: 1300, wantOK: true, wantAmount: 100, wantTotal: 1200,
		},
		{
			name:       "нет товаров категории",
			category:   "Книги",
			items:      []models.CartItem{cartItem(500, 5, "Футболки")},
			itemsTotal: 2500, wantOK: false, wantAmount: 0, wantTotal: 2500,
		},
		{
			name:       "пустая корзина",
			category:   "Книги",
			items:      []models.CartItem{},
			itemsTotal: 0, wantOK: false, wantAmount: 0, wantTotal: 0,
		},
		{
			name:       "quantity > 1 разворачивается",
			category:   "Книги",
			items:      []models.CartItem{cartItem(100, 4, "Книги")},
			itemsTotal: 400, wantOK: true, wantAmount: 100, wantTotal: 300, // 4 единицы, 1 бесплатная
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ctxWithItems3(tt.items, tt.itemsTotal)
			d := NewThreeForTwo(tt.category, "тест")

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
