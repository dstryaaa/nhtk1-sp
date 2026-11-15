package models

import "time"

type Product struct {
	Id        int
	Name      string
	BasePrice int64 // в копейках
	Category  string
}

type CartItem struct {
	Product  *Product
	Quantity int
}

type Customer struct {
	Id               string
	LastPurchaseDate *time.Time // nil — покупок не было
	IsFirstOrder     bool
}

type Order struct {
	Id           string
	Customer     Customer
	Items        []CartItem
	PromoCode    *string // nil — код не вводили
	CreatedAt    time.Time
	DeliveryCost int64
}

type AppliedDiscount struct {
	Name   string
	Amount int64
	Reason string
}

type PricingResult struct {
	BaseTotal        int64
	AppliedDiscounts []AppliedDiscount
	FinalTotal       int64
}

// DiscountContext — единственный изменяемый объект за расчёт.
// ItemsTotal и DeliveryCost ведём отдельно: скидки на товары не должны трогать доставку.
type DiscountContext struct {
	Order        *Order
	ItemsTotal   int64
	DeliveryCost int64
	Applied      []AppliedDiscount

	PromoCode    *string
	PromoValid   bool
	PromoApplied bool
}
