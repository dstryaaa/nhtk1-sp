package models

import "time"

type Product struct {
	Id        int
	Name      string
	BasePrice int64
	Category  string
}

type CartItem struct {
	Product  *Product
	Quantity int
}

type Customer struct {
	Id               string
	LastPurchaseDate *time.Time
	IsFirstOrder     bool
}

type Order struct {
	Id           string
	Customer     Customer
	Items        []CartItem
	PromoCode    *string
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

type DiscountContext struct {
	Order        *Order
	ItemsTotal   int64
	DeliveryCost int64
	Applied      []AppliedDiscount
	PromoCode    *string
	PromoValid   bool
	PromoApplied bool
}
