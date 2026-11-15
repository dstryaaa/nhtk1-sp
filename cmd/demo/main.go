package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dstryaaa/nhtk1/internal/engine"
	"github.com/dstryaaa/nhtk1/internal/factory"
	"github.com/dstryaaa/nhtk1/internal/models"
)

func main() {
	// Фиксируем даты, чтобы демо было воспроизводимым.
	created := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	lastPurchase := created.AddDate(0, 0, -12)
	promo := "SAVE15"

	// Все деньги — в копейках: 50000 = 500 ₽.
	cfg := factory.Config{
		Discounts: []factory.DiscountConfig{
			{Type: "threeForTwo", Category: "Книги", Name: "3 по цене 2"},
			{Type: "percent", Name: "Осенняя распродажа", Percent: 10},
			{Type: "loyalty", Name: "Лояльность", Percent: 5, Days: 30},
			{Type: "fixed", Name: "500 рублей от 3000", Value: 50000, Threshold: 300000},
			{Type: "freeDelivery", Name: "Бесплатная доставка", Threshold: 500000},
		},
		PromoCodes: []factory.PromoConfig{
			{Code: "SAVE15", Name: "Промокод 15%", Type: "percent", Percent: 15, Active: true},
		},
	}

	e, err := engine.NewPricingEngine(cfg)
	if err != nil {
		panic(err)
	}

	o := &models.Order{
		Customer: models.Customer{
			IsFirstOrder:     false,
			LastPurchaseDate: &lastPurchase,
		},
		Items: []models.CartItem{
			{Product: &models.Product{BasePrice: 100000, Category: "Книги"}, Quantity: 6},
		},
		DeliveryCost: 50000,
		PromoCode:    &promo,
		CreatedAt:    created,
	}

	res, err := e.Calculate(o)
	if err != nil {
		panic(err)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
