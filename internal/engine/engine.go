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

	// Проверка промокода
	var promoCfg *factory.PromoConfig
	var promoObj *discounts.PromoCode
	if ctx.PromoCode != nil {
		if cfg, ok := factory.FindPromo(e.config, *ctx.PromoCode); ok {
			if cfg.Active && (cfg.ValidUntil == nil || !order.CreatedAt.After(*cfg.ValidUntil)) {
				ctx.PromoValid = true
				promoCfg = cfg
				promoObj, _ = factory.CreatePromo(*cfg)
			}
		}
	}

	// 2. ThreeForTwo
	for _, d := range e.strategies.threeForTwo {
		if ad, ok := d.Apply(ctx); ok {
			ctx.Applied = append(ctx.Applied, ad)
		}
	}

	// 3. Percent — с парой "обычные процентные vs процентный промокод"
	var pairCandidates []discounts.Discount
	var outsidePair []discounts.Discount
	for _, d := range e.strategies.percent {
		if d.Kind() == discounts.KindPercent {
			pairCandidates = append(pairCandidates, d)
		} else {
			outsidePair = append(outsidePair, d)
		}
	}

	// Симуляция обычных процентных на копии контекста.
	simReg := *ctx
	simReg.Applied = append([]models.AppliedDiscount(nil), ctx.Applied...)
	for _, d := range pairCandidates {
		d.Apply(&simReg)
	}
	savedRegular := ctx.ItemsTotal - simReg.ItemsTotal

	// Симуляция процентного промокода (если он валиден и процентный).
	var savedPromo int64
	if ctx.PromoValid && promoCfg.Type == "percent" {
		simPromo := *ctx
		simPromo.Applied = append([]models.AppliedDiscount(nil), ctx.Applied...)
		promoObj.Apply(&simPromo)
		savedPromo = ctx.ItemsTotal - simPromo.ItemsTotal
	}

	// Выбор победителя: больше выгода; при равенстве — промокод.
	if savedPromo > 0 && savedPromo >= savedRegular {
		if ad, ok := promoObj.Apply(ctx); ok {
			ctx.Applied = append(ctx.Applied, ad)
			ctx.PromoApplied = true
		}
	} else {
		for _, d := range pairCandidates {
			if ad, ok := d.Apply(ctx); ok {
				ctx.Applied = append(ctx.Applied, ad)
			}
		}
	}

	// Лояльность и первый заказ — вне пары, после неё.
	for _, d := range outsidePair {
		if ad, ok := d.Apply(ctx); ok {
			ctx.Applied = append(ctx.Applied, ad)
		}
	}

	//4. Fixed
	for _, d := range e.strategies.fixed {
		if ad, ok := d.Apply(ctx); ok {
			ctx.Applied = append(ctx.Applied, ad)
		}
	}
	// Фикс-промокод — на стадии Fixed, после обычных.
	if ctx.PromoValid && promoCfg.Type == "fixed" && !ctx.PromoApplied {
		if ad, ok := promoObj.Apply(ctx); ok {
			ctx.Applied = append(ctx.Applied, ad)
			ctx.PromoApplied = true
		}
	}

	// 5. Delivery
	for _, d := range e.strategies.delivery {
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
