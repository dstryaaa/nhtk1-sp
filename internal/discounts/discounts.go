package discounts

import (
	"fmt"
	"slices"
	"time"

	"github.com/dstryaaa/nhtk1/internal/models"
)

type Kind int

const (
	KindPercent Kind = iota
	KindFixed
	KindThreeForTwo
	KindLoyalty
	KindFirstOrder
	KindFreeDelivery
	KindPromoCode
)

type Discount interface {
	Name() string
	Kind() Kind
	Apply(ctx *models.DiscountContext) (models.AppliedDiscount, bool)
}

type PercentDiscount struct {
	name    string
	percent int64
}

func NewPercentDiscount(name string, percent int64) *PercentDiscount {
	return &PercentDiscount{name: name, percent: percent}
}

func (d *PercentDiscount) Name() string { return d.name }
func (d *PercentDiscount) Kind() Kind   { return KindPercent }

func (d *PercentDiscount) Apply(ctx *models.DiscountContext) (models.AppliedDiscount, bool) {
	if d.percent <= 0 || ctx.ItemsTotal <= 0 {
		return models.AppliedDiscount{}, false
	}

	oldTotal := ctx.ItemsTotal

	// Считаем остаток, а не саму скидку: так при округлении old - new всегда сойдётся.
	newTotal := roundToHalfEven(oldTotal*(100-d.percent), 100)
	if newTotal < 0 {
		newTotal = 0
	}

	amount := oldTotal - newTotal
	if amount <= 0 {
		return models.AppliedDiscount{}, false
	}

	ctx.ItemsTotal = newTotal
	return models.AppliedDiscount{
		Name:   d.name,
		Amount: amount,
		Reason: fmt.Sprintf("Процентная скидка %d%% на товары", d.percent),
	}, true
}

type FixedDiscount struct {
	value     int64
	threshold int64
	name      string
}

func NewFixedDiscount(value int64, threshold int64, name string) *FixedDiscount {
	return &FixedDiscount{value: value, threshold: threshold, name: name}
}

func (d *FixedDiscount) Name() string { return d.name }
func (d *FixedDiscount) Kind() Kind   { return KindFixed }

func (d *FixedDiscount) Apply(ctx *models.DiscountContext) (models.AppliedDiscount, bool) {
	if d.value <= 0 || ctx.ItemsTotal <= 0 {
		return models.AppliedDiscount{}, false
	}
	// Порог проверяем до скидки, «не меньше» — значит >=.
	if ctx.ItemsTotal < d.threshold {
		return models.AppliedDiscount{}, false
	}

	// Ниже нуля не уходим — режем скидку по текущей сумме.
	amount := min(d.value, ctx.ItemsTotal)
	ctx.ItemsTotal -= amount
	return models.AppliedDiscount{
		Name:   d.name,
		Amount: amount,
		Reason: fmt.Sprintf("Фиксированная скидка %.2f руб. на товары", float64(amount)/100),
	}, true
}

type FreeDelivery struct {
	threshold int64
	name      string
}

func NewFreeDelivery(threshold int64, name string) *FreeDelivery {
	return &FreeDelivery{threshold: threshold, name: name}
}

func (d *FreeDelivery) Name() string { return d.name }
func (d *FreeDelivery) Kind() Kind   { return KindFreeDelivery }

func (d *FreeDelivery) Apply(ctx *models.DiscountContext) (models.AppliedDiscount, bool) {
	// Строго больше — ровно на пороге доставка ещё платная.
	if ctx.ItemsTotal <= d.threshold {
		return models.AppliedDiscount{}, false
	}
	if ctx.DeliveryCost == 0 {
		return models.AppliedDiscount{}, false
	}

	amount := ctx.DeliveryCost
	ctx.DeliveryCost = 0
	return models.AppliedDiscount{
		Name:   d.name,
		Amount: amount,
		Reason: fmt.Sprintf("Бесплатная доставка при сумме товаров больше %d коп.", d.threshold),
	}, true
}

type ThreeForTwo struct {
	category string
	name     string
}

func NewThreeForTwo(category string, name string) *ThreeForTwo {
	return &ThreeForTwo{category: category, name: name}
}

func (d *ThreeForTwo) Name() string { return d.name }
func (d *ThreeForTwo) Kind() Kind   { return KindThreeForTwo }

func (d *ThreeForTwo) Apply(ctx *models.DiscountContext) (models.AppliedDiscount, bool) {
	// Разворачиваем quantity в отдельные единицы — «3 по 2» считает штуки, а не позиции.
	unitPrices := make([]int64, 0)
	for _, item := range ctx.Order.Items {
		if item.Product.Category != d.category || item.Quantity <= 0 {
			continue
		}
		for i := 0; i < item.Quantity; i++ {
			unitPrices = append(unitPrices, item.Product.BasePrice)
		}
	}

	totalQuantity := len(unitPrices)
	freeCount := totalQuantity / 3
	if freeCount == 0 {
		return models.AppliedDiscount{}, false
	}

	// Бесплатными становятся самые дешёвые единицы — сортируем и берём первые freeCount.
	slices.Sort(unitPrices)
	var amount int64
	for i := 0; i < freeCount; i++ {
		amount += unitPrices[i]
	}
	if amount <= 0 {
		return models.AppliedDiscount{}, false
	}

	if amount < ctx.ItemsTotal {
		ctx.ItemsTotal -= amount
	}

	return models.AppliedDiscount{
		Name:   d.name,
		Amount: amount,
		Reason: fmt.Sprintf("Категория %s: %d единиц, %d бесплатно", d.category, totalQuantity, freeCount),
	}, true
}

type FirstOrder struct {
	name  string
	inner *PercentDiscount
}

func NewFirstOrder(name string, percent int64) *FirstOrder {
	return &FirstOrder{name: name, inner: NewPercentDiscount(name, percent)}
}

func (d *FirstOrder) Name() string { return d.name }
func (d *FirstOrder) Kind() Kind   { return KindFirstOrder }

func (d *FirstOrder) Apply(ctx *models.DiscountContext) (models.AppliedDiscount, bool) {
	if !ctx.Order.Customer.IsFirstOrder {
		return models.AppliedDiscount{}, false
	}

	res, ok := d.inner.Apply(ctx)
	if !ok {
		return models.AppliedDiscount{}, false
	}
	// inner знает только про проценты — имя и причину подменяем на свои.
	res.Name = d.name
	res.Reason = "Скидка за первый заказ"
	return res, true
}

type Loyalty struct {
	name  string
	days  int
	inner *PercentDiscount
}

func NewLoyalty(name string, days int, percent int64) *Loyalty {
	return &Loyalty{name: name, days: days, inner: NewPercentDiscount(name, percent)}
}

func (d *Loyalty) Name() string { return d.name }
func (d *Loyalty) Kind() Kind   { return KindLoyalty }

func (d *Loyalty) Apply(ctx *models.DiscountContext) (models.AppliedDiscount, bool) {
	last := ctx.Order.Customer.LastPurchaseDate
	if last == nil {
		return models.AppliedDiscount{}, false
	}

	// Считаем от даты заказа, а не от time.Now() — иначе тесты будут плавать.
	days, ok := daysBetween(*last, ctx.Order.CreatedAt)
	if !ok {
		// Дата в будущем — данные битые, скидку не даём.
		return models.AppliedDiscount{}, false
	}
	if days > d.days {
		return models.AppliedDiscount{}, false
	}

	res, ok := d.inner.Apply(ctx)
	if !ok {
		return models.AppliedDiscount{}, false
	}

	res.Name = d.name
	res.Reason = fmt.Sprintf("Лояльность: последняя покупка %d дней назад", days)
	return res, true
}

type PromoCode struct {
	name       string
	code       string
	validUntil *time.Time
	active     bool
	inner      Discount
}

func NewPromoCode(name string, code string, validUntil *time.Time, active bool, inner Discount) *PromoCode {
	return &PromoCode{name: name, code: code, validUntil: validUntil, active: active, inner: inner}
}

func (pc *PromoCode) Name() string           { return pc.name }
func (pc *PromoCode) Code() string           { return pc.code }
func (pc *PromoCode) ValidUntil() *time.Time { return pc.validUntil }
func (pc *PromoCode) Active() bool           { return pc.active }
func (pc *PromoCode) Kind() Kind             { return KindPromoCode }

func (pc *PromoCode) Apply(ctx *models.DiscountContext) (models.AppliedDiscount, bool) {
	// PromoApplied ставит движок — здесь только читаем, иначе симуляция на копии всё сломает.
	if ctx.PromoApplied || !ctx.PromoValid {
		return models.AppliedDiscount{}, false
	}

	res, ok := pc.inner.Apply(ctx)
	if !ok {
		return models.AppliedDiscount{}, false
	}

	res.Name = pc.name
	res.Reason = fmt.Sprintf("Промокод %s: скидка %d.%02d руб.", pc.code, res.Amount/100, res.Amount%100)
	return res, true
}

// roundToHalfEven — банковское округление a/b. Отрицательные не поддерживаем: суммы у нас неотрицательные.
func roundToHalfEven(a, b int64) int64 {
	q := a / b
	r := a % b
	if r*2 > b {
		q++
	} else if r*2 == b {
		// Ровно половина — к чётному.
		if q%2 == 1 {
			q++
		}
	}
	return q
}

// daysBetween — целые календарные дни между датами. Обе даты должны быть в одной таймзоне.
func daysBetween(from, to time.Time) (int, bool) {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location())

	days := int(to.Sub(from).Hours() / 24)
	if days < 0 {
		// from позже to — будущая дата, считаем некорректной.
		return 0, false
	}
	return days, true
}
