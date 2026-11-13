package engine

import (
	"fmt"

	"github.com/dstryaaa/nhtk1/internal/discounts"
	"github.com/dstryaaa/nhtk1/internal/factory"
	"github.com/dstryaaa/nhtk1/internal/models"
)

type Stage int

const (
	StageThreeForTwo Stage = iota
	StagePercent
	StageFixed
	StageDelivery
)

type strategies struct {
	threeForTwo []discounts.Discount
	percent     []discounts.Discount
	fixed       []discounts.Discount
	delivery    []discounts.Discount
}

type PricingEngine struct {
	config     factory.Config
	strategies strategies
}

func NewPricingEngine(config factory.Config) (*PricingEngine, error) {
	var s strategies
	for _, cfg := range config.Discounts {
		d, err := factory.Create(cfg)
		if err != nil {
			return nil, err
		}
		switch d.Kind() {
		case discounts.KindThreeForTwo:
			s.threeForTwo = append(s.threeForTwo, d)
		case discounts.KindPercent, discounts.KindLoyalty, discounts.KindFirstOrder:
			s.percent = append(s.percent, d)
		case discounts.KindFixed:
			s.fixed = append(s.fixed, d)
		case discounts.KindFreeDelivery:
			s.delivery = append(s.delivery, d)
		default:
			return nil, fmt.Errorf("unsupported kind: %v", d.Kind())
		}
	}
	return &PricingEngine{
		config:     config,
		strategies: s,
	}, nil
}

func (e *PricingEngine) Calculate(order *models.Order) (*models.PricingResult, error) {
	if order == nil {
		return nil, fmt.Errorf("отсутствует заказ")
	}

	for _, item := range order.Items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("некорректное quantity: %d", item.Quantity)
		}
	}
	// 1. Начальное состояние контекста.
	itemsSum := sumItems(order.Items)

	ctx := &models.DiscountContext{
		Order:        order,
		ItemsTotal:   itemsSum,
		DeliveryCost: order.DeliveryCost,
		Applied:      make([]models.AppliedDiscount, 0),
		PromoCode:    order.PromoCode,
	}

	// 2. ThreeForTwo
	for _, d := range e.strategies.threeForTwo {
		if ad, ok := d.Apply(ctx); ok {
			ctx.Applied = append(ctx.Applied, ad)
		}
	}

	return &models.PricingResult{
		BaseTotal:        itemsSum + order.DeliveryCost,
		AppliedDiscounts: ctx.Applied,
		FinalTotal:       ctx.ItemsTotal + ctx.DeliveryCost,
	}, nil
}

func sumItems(items []models.CartItem) int64 {
	res := int64(0)
	for _, item := range items {
		res += int64(item.Quantity) * item.Product.BasePrice
	}
	return res
}
