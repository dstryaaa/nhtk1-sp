package engine

import (
	"fmt"

	"github.com/dstryaaa/nhtk1/internal/discounts"
	"github.com/dstryaaa/nhtk1/internal/factory"
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
		case discounts.KindPercent:
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
