package factory

import (
	"fmt"
	"time"

	"github.com/dstryaaa/nhtk1/internal/discounts"
)

type DiscountConfig struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Value     int64  `json:"value"`
	Threshold int64  `json:"threshold"`
	Category  string `json:"category"`
	Percent   int64  `json:"percent"`
	Days      int    `json:"days"`
}

type PromoConfig struct {
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Value      int64      `json:"value"`
	Percent    int64      `json:"percent"`
	ValidUntil *time.Time `json:"valid_until"`
	Active     bool       `json:"active"`
}

type Config struct {
	Discounts  []DiscountConfig `json:"discounts"`
	PromoCodes []PromoConfig    `json:"promo_codes"`
}

// Create — единственное место, где по Type решается, какой класс создавать.
func Create(cfg DiscountConfig) (discounts.Discount, error) {
	switch cfg.Type {
	case "percent":
		return discounts.NewPercentDiscount(cfg.Name, cfg.Percent), nil
	case "fixed":
		return discounts.NewFixedDiscount(cfg.Value, cfg.Threshold, cfg.Name), nil
	case "freeDelivery":
		return discounts.NewFreeDelivery(cfg.Threshold, cfg.Name), nil
	case "threeForTwo":
		return discounts.NewThreeForTwo(cfg.Category, cfg.Name), nil
	case "firstOrder":
		return discounts.NewFirstOrder(cfg.Name, cfg.Percent), nil
	case "loyalty":
		return discounts.NewLoyalty(cfg.Name, cfg.Days, cfg.Percent), nil
	default:
		return nil, fmt.Errorf("unknown discount type: %q", cfg.Type)
	}
}

// CreatePromo возвращает *PromoCode: движку нужны Code/ValidUntil/Active
// для проверки валидности до применения.
func CreatePromo(cfg PromoConfig) (*discounts.PromoCode, error) {
	var inner discounts.Discount
	switch cfg.Type {
	case "percent":
		inner = discounts.NewPercentDiscount(cfg.Name, cfg.Percent)
	case "fixed":
		// У промокода нет порога — 0, чтобы срабатывал всегда.
		inner = discounts.NewFixedDiscount(cfg.Value, 0, cfg.Name)
	default:
		return nil, fmt.Errorf("unknown promocode type: %q", cfg.Type)
	}
	return discounts.NewPromoCode(cfg.Name, cfg.Code, cfg.ValidUntil, cfg.Active, inner), nil
}

// FindPromo ищет промокод по коду. Конфиг не мутируем — указатель на элемент безопасен.
func FindPromo(cfg Config, code string) (*PromoConfig, bool) {
	for i := range cfg.PromoCodes {
		if cfg.PromoCodes[i].Code == code {
			return &cfg.PromoCodes[i], true
		}
	}
	return nil, false
}
