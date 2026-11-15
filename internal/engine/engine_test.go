package engine

import (
	"testing"
	"time"

	"github.com/dstryaaa/nhtk1/internal/factory"
	"github.com/dstryaaa/nhtk1/internal/models"
)

// --- Хелперы ---

func item(price int64, qty int, category string) models.CartItem {
	return models.CartItem{
		Product:  &models.Product{BasePrice: price, Category: category},
		Quantity: qty,
	}
}

func order(items []models.CartItem, delivery int64, customer models.Customer, promo *string, createdAt time.Time) *models.Order {
	return &models.Order{
		Customer:     customer,
		Items:        items,
		DeliveryCost: delivery,
		PromoCode:    promo,
		CreatedAt:    createdAt,
	}
}

func ptrStr(s string) *string        { return &s }
func ptrTime(t time.Time) *time.Time { return &t }

func mustEngine(t *testing.T, cfg factory.Config) *PricingEngine {
	t.Helper()
	e, err := NewPricingEngine(cfg)
	if err != nil {
		t.Fatalf("NewPricingEngine: %v", err)
	}
	return e
}

func findDiscount(res *models.PricingResult, name string) (models.AppliedDiscount, bool) {
	for _, ad := range res.AppliedDiscounts {
		if ad.Name == name {
			return ad, true
		}
	}
	return models.AppliedDiscount{}, false
}

// --- 1. 3 по цене 2 + процент: количественная первой, процент от остатка ---

func TestEngine_ThreeForTwoThenPercent(t *testing.T) {
	// 6 книг по 100 = 600. 3по2: 2 бесплатные = 200, остаток 400.
	// Процент 10% от 400 = 40. Итог 360.
	cfg := factory.Config{Discounts: []factory.DiscountConfig{
		{Type: "threeForTwo", Category: "Книги", Name: "3 по 2"},
		{Type: "percent", Name: "10%", Percent: 10},
	}}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{item(100, 6, "Книги")}, 0, models.Customer{}, nil, time.Now())

	res, err := e.Calculate(o)
	if err != nil {
		t.Fatal(err)
	}
	if res.BaseTotal != 600 {
		t.Errorf("BaseTotal = %d, want 600", res.BaseTotal)
	}
	if res.FinalTotal != 360 {
		t.Errorf("FinalTotal = %d, want 360", res.FinalTotal)
	}
	// Порядок breakdown: 3по2 первым, процент вторым.
	if len(res.AppliedDiscounts) != 2 {
		t.Fatalf("len(Applied) = %d, want 2", len(res.AppliedDiscounts))
	}
	if res.AppliedDiscounts[0].Name != "3 по 2" || res.AppliedDiscounts[0].Amount != 200 {
		t.Errorf("первая скидка = %+v, want 3 по 2 / 200", res.AppliedDiscounts[0])
	}
	if res.AppliedDiscounts[1].Name != "10%" || res.AppliedDiscounts[1].Amount != 40 {
		t.Errorf("вторая скидка = %+v, want 10%% / 40", res.AppliedDiscounts[1])
	}
}

// --- 2. Промокод vs процент: большая выгода; равенство → промокод ---

func TestEngine_PromoVsPercent_BiggerWins(t *testing.T) {
	// ItemsTotal 1000. Обычная 10% = 100. Промокод 15% = 150. Побеждает промокод.
	created := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	cfg := factory.Config{
		Discounts: []factory.DiscountConfig{
			{Type: "percent", Name: "Обычная 10%", Percent: 10},
		},
		PromoCodes: []factory.PromoConfig{
			{Code: "SAVE15", Name: "Промокод 15%", Type: "percent", Percent: 15, Active: true},
		},
	}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{item(1000, 1, "X")}, 0, models.Customer{}, ptrStr("SAVE15"), created)

	res, err := e.Calculate(o)
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalTotal != 850 {
		t.Errorf("FinalTotal = %d, want 850 (промокод 150)", res.FinalTotal)
	}
	if _, ok := findDiscount(res, "Промокод 15%"); !ok {
		t.Error("в breakdown нет промокода — он должен победить")
	}
	if _, ok := findDiscount(res, "Обычная 10%"); ok {
		t.Error("обычная 10% не должна применяться — промокод победил")
	}
}

func TestEngine_PromoVsPercent_TiePromoWins(t *testing.T) {
	// ItemsTotal 1000. Обычная 10% = 100. Промокод 10% = 100. Равенство → промокод.
	created := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	cfg := factory.Config{
		Discounts: []factory.DiscountConfig{
			{Type: "percent", Name: "Обычная 10%", Percent: 10},
		},
		PromoCodes: []factory.PromoConfig{
			{Code: "TIE10", Name: "Промокод 10%", Type: "percent", Percent: 10, Active: true},
		},
	}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{item(1000, 1, "X")}, 0, models.Customer{}, ptrStr("TIE10"), created)

	res, _ := e.Calculate(o)
	if _, ok := findDiscount(res, "Промокод 10%"); !ok {
		t.Error("при равенстве должен победить промокод")
	}
	if _, ok := findDiscount(res, "Обычная 10%"); ok {
		t.Error("обычная не должна применяться при равенстве")
	}
}

// --- 3. Нижняя граница: фикс не уводит ниже 0 ---

func TestEngine_FixedDoesNotGoBelowZero(t *testing.T) {
	// ItemsTotal 100, фикс 5000. Скидка урезается до 100, итог 0.
	cfg := factory.Config{Discounts: []factory.DiscountConfig{
		{Type: "fixed", Name: "5000", Value: 5000, Threshold: 0},
	}}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{item(100, 1, "X")}, 0, models.Customer{}, nil, time.Now())

	res, _ := e.Calculate(o)
	if res.FinalTotal != 0 {
		t.Errorf("FinalTotal = %d, want 0", res.FinalTotal)
	}
	ad, ok := findDiscount(res, "5000")
	if !ok {
		t.Fatal("скидка должна быть в breakdown")
	}
	if ad.Amount != 100 {
		t.Errorf("Amount = %d, want 100 (урезано)", ad.Amount)
	}
}

// --- 4. Округление: ровно половина копейки — half-to-even ---

func TestEngine_RoundingHalfToEven(t *testing.T) {
	// ItemsTotal 1 копейка, 50% → 0.5 → half-to-even → 0 (чётное).
	// Amount = 1 - 0 = 1. FinalTotal = 0.
	cfg := factory.Config{Discounts: []factory.DiscountConfig{
		{Type: "percent", Name: "50%", Percent: 50},
	}}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{item(1, 1, "X")}, 0, models.Customer{}, nil, time.Now())

	res, _ := e.Calculate(o)
	if res.FinalTotal != 0 {
		t.Errorf("FinalTotal = %d, want 0", res.FinalTotal)
	}
	ad, _ := findDiscount(res, "50%")
	if ad.Amount != 1 {
		t.Errorf("Amount = %d, want 1", ad.Amount)
	}
}

// --- 5. Границы ---

func TestEngine_Boundaries(t *testing.T) {
	created := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("порог фикса включителен", func(t *testing.T) {
		cfg := factory.Config{Discounts: []factory.DiscountConfig{
			{Type: "fixed", Name: "фикс", Value: 500, Threshold: 3000},
		}}
		e := mustEngine(t, cfg)
		o := order([]models.CartItem{item(3000, 1, "X")}, 0, models.Customer{}, nil, created)
		res, _ := e.Calculate(o)
		if res.FinalTotal != 2500 {
			t.Errorf("при ровно пороге скидка должна применяться: FinalTotal = %d", res.FinalTotal)
		}
	})

	t.Run("ровно 30 дней лояльности", func(t *testing.T) {
		cfg := factory.Config{Discounts: []factory.DiscountConfig{
			{Type: "loyalty", Name: "лояльность", Percent: 5, Days: 30},
		}}
		e := mustEngine(t, cfg)
		o := order([]models.CartItem{item(1000, 1, "X")}, 0,
			models.Customer{LastPurchaseDate: ptrTime(created.AddDate(0, 0, -30))}, nil, created)
		res, _ := e.Calculate(o)
		if res.FinalTotal != 950 {
			t.Errorf("ровно 30 дней должно подходить: FinalTotal = %d", res.FinalTotal)
		}
	})

	t.Run("дата окончания промокода включительна", func(t *testing.T) {
		cfg := factory.Config{
			PromoCodes: []factory.PromoConfig{
				{Code: "END", Name: "промо", Type: "percent", Percent: 10, Active: true, ValidUntil: ptrTime(created)},
			},
		}
		e := mustEngine(t, cfg)
		o := order([]models.CartItem{item(1000, 1, "X")}, 0, models.Customer{}, ptrStr("END"), created)
		res, _ := e.Calculate(o)
		if _, ok := findDiscount(res, "промо"); !ok {
			t.Error("в день окончания промокод должен быть валиден")
		}
	})

	t.Run("доставка строго больше", func(t *testing.T) {
		cfg := factory.Config{Discounts: []factory.DiscountConfig{
			{Type: "freeDelivery", Name: "бесплатная доставка", Threshold: 5000},
		}}
		e := mustEngine(t, cfg)
		// ровно 5000 — не применяется
		o1 := order([]models.CartItem{item(5000, 1, "X")}, 500, models.Customer{}, nil, created)
		res1, _ := e.Calculate(o1)
		if _, ok := findDiscount(res1, "бесплатная доставка"); ok {
			t.Error("при ровно пороге доставка не должна обнуляться")
		}
		// 5001 — применяется
		o2 := order([]models.CartItem{item(5001, 1, "X")}, 500, models.Customer{}, nil, created)
		res2, _ := e.Calculate(o2)
		if _, ok := findDiscount(res2, "бесплатная доставка"); !ok {
			t.Error("при 5001 доставка должна обнулиться")
		}
	})
}

// --- 6. Breakdown: названия, суммы, причины, порядок ---

func TestEngine_Breakdown(t *testing.T) {
	cfg := factory.Config{Discounts: []factory.DiscountConfig{
		{Type: "percent", Name: "Скидка A", Percent: 10},
		{Type: "fixed", Name: "Скидка B", Value: 100, Threshold: 0},
	}}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{item(1000, 1, "X")}, 0, models.Customer{}, nil, time.Now())

	res, _ := e.Calculate(o)
	if len(res.AppliedDiscounts) != 2 {
		t.Fatalf("len = %d, want 2", len(res.AppliedDiscounts))
	}
	if res.AppliedDiscounts[0].Name != "Скидка A" {
		t.Errorf("первая должна быть A (конфиг-порядок)")
	}
	if res.AppliedDiscounts[1].Name != "Скидка B" {
		t.Errorf("вторая должна быть B")
	}
	for _, ad := range res.AppliedDiscounts {
		if ad.Reason == "" {
			t.Errorf("Reason пустой у %q", ad.Name)
		}
		if ad.Amount <= 0 {
			t.Errorf("Amount должен быть > 0 у %q", ad.Name)
		}
	}
}

// --- Рекомендуемые: пустая корзина, неизвестный промокод, все совместимые ---

func TestEngine_EmptyCart(t *testing.T) {
	cfg := factory.Config{Discounts: []factory.DiscountConfig{
		{Type: "percent", Name: "10%", Percent: 10},
	}}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{}, 300, models.Customer{}, nil, time.Now())
	res, err := e.Calculate(o)
	if err != nil {
		t.Fatal(err)
	}
	if res.BaseTotal != 300 || res.FinalTotal != 300 {
		t.Errorf("пустая корзина: BaseTotal=%d FinalTotal=%d, want 300/300", res.BaseTotal, res.FinalTotal)
	}
}

func TestEngine_UnknownPromo(t *testing.T) {
	cfg := factory.Config{}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{item(1000, 1, "X")}, 0, models.Customer{}, ptrStr("NOPE"), time.Now())
	res, _ := e.Calculate(o)
	if len(res.AppliedDiscounts) != 0 {
		t.Error("неизвестный промокод не должен применяться")
	}
}

func TestEngine_ExpiredPromo(t *testing.T) {
	created := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	cfg := factory.Config{PromoCodes: []factory.PromoConfig{
		{Code: "OLD", Name: "старый", Type: "percent", Percent: 10, Active: true,
			ValidUntil: ptrTime(created.AddDate(0, 0, -1))}, // истёк вчера
	}}
	e := mustEngine(t, cfg)
	o := order([]models.CartItem{item(1000, 1, "X")}, 0, models.Customer{}, ptrStr("OLD"), created)
	res, _ := e.Calculate(o)
	if _, ok := findDiscount(res, "старый"); ok {
		t.Error("просроченный промокод не должен применяться")
	}
}

func TestEngine_AllCompatible(t *testing.T) {
	// 3по2 + процент + лояльность + первый заказ + фикс + доставка.
	created := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	cfg := factory.Config{Discounts: []factory.DiscountConfig{
		{Type: "threeForTwo", Category: "Книги", Name: "3 по 2"},
		{Type: "percent", Name: "10%", Percent: 10},
		{Type: "loyalty", Name: "лояльность", Percent: 5, Days: 30},
		{Type: "firstOrder", Name: "первый", Percent: 10},
		{Type: "fixed", Name: "фикс", Value: 100, Threshold: 0},
		{Type: "freeDelivery", Name: "доставка", Threshold: 1000},
	}}
	e := mustEngine(t, cfg)
	o := order(
		[]models.CartItem{item(100, 6, "Книги")}, // 600, 3по2 → 200, остаток 400
		500,
		models.Customer{IsFirstOrder: true, LastPurchaseDate: ptrTime(created.AddDate(0, 0, -10))},
		nil, created,
	)
	res, err := e.Calculate(o)
	if err != nil {
		t.Fatal(err)
	}
	// Порядок: 3по2, потом процент/лояльность/первый (все совместимы), фикс, доставка.
	// Точные суммы зависят от порядка лояльности/первого — проверим инварианты.
	if res.BaseTotal != 600+500 {
		t.Errorf("BaseTotal = %d, want 1100", res.BaseTotal)
	}
	if res.FinalTotal < 0 {
		t.Errorf("FinalTotal < 0: %d", res.FinalTotal)
	}
	// Все 6 скидок должны быть в breakdown (доставка тоже — ItemsTotal после скидок > 1000? проверим).
	names := map[string]bool{}
	for _, ad := range res.AppliedDiscounts {
		names[ad.Name] = true
	}
	for _, want := range []string{"3 по 2", "10%", "лояльность", "первый", "фикс"} {
		if !names[want] {
			t.Errorf("в breakdown нет %q", want)
		}
	}
}
