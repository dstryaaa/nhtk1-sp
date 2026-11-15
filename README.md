# Движок скидок «Штучки-Дрючки»

## Запуск

```bash
go test ./...
go run ./cmd/demo
```

`-race` на Windows требует cgo. В CI/Linux: `CGO_ENABLED=1 go test ./... -race`.

## Структура

```
cmd/demo/          — демонстрационный пример
internal/
  models/          — Product, CartItem, Customer, Order, DiscountContext, AppliedDiscount, PricingResult
  discounts/       — интерфейс Discount + 7 стратегий + roundToHalfEven + daysBetween
  factory/         — Create / CreatePromo / FindPromo (единственный switch по Type)
  engine/          — PricingEngine: оркестратор стадий и пары промокод/процент
  provider/        — ConfigProvider (Singleton, только конфиг)
```

## Архитектура

- Все стратегии реализуют `Discount { Name(); Kind(); Apply(ctx) (AppliedDiscount, bool) }`.
- Стратегия мутирует `DiscountContext` (`ItemsTotal` или `DeliveryCost`) и возвращает `AppliedDiscount`; движок решает, добавлять ли его в breakdown.
- `PricingEngine` не содержит `if type == ...` — только `switch` в фабрике.
- `ConfigProvider` хранит только `Config`, без состояния заказа.

## Порядок расчёта

1. ThreeForTwo — до денежных скидок.
2. Percent — пара «обычные процентные vs процентный промокод»: симуляция на копии контекста, сравнение выгоды на одной сумме, при равенстве побеждает промокод. Лояльность и первый заказ — после пары, в порядке конфига.
3. Fixed — обычные фиксированные, затем фикс-промокод (если валиден).
4. Delivery — бесплатная доставка (строго больше порога).

## Округление и деньги

- Деньги — `int64` в копейках (не `float64`: IEEE-754, flaky-тесты).
- Half-to-even при делении на 100 (`roundToHalfEven`).
- `amount = old - new` — баланс `ItemsTotal` и `Amount` всегда сходится.
- В breakdown попадают только скидки с `Amount > 0`.

## Промокоды

- Валидность проверяет движок до стадий: `active && (validUntil == nil || CreatedAt <= *validUntil)` (срок включителен, сравнение с `CreatedAt`, не с `time.Now()`).
- Процентный — в паре с обычными процентными; при равенстве побеждает.
- Фиксированный — на стадии Fixed после обычных.
- `PromoApplied` ставит движок (иначе симуляция на копии «насорила» бы состоянием).

## Принятые решения

- `int64` в копейках везде, включая конфиг (демо: `Value: 50000` = 500 ₽).
- `ThreeForTwo`: глобальная сортировка единиц категории, первые `floor(qty/3)` — бесплатные (на `3×100 + 1×50` бесплатная 50, не 100).
- `daysBetween`: календарные дни через `time.Date` с локальной `Location`, без `UTC()`; 30 дней включительно; будущая дата → `false`.
- `Quantity <= 0` → ошибка из `Calculate`.
- Конфиг неизменяем после `SetConfig`.
- `PromoCode` создаётся как `*PromoCode` (движку нужны `Code/ValidUntil/Active` для проверки до применения); применение — через `Discount.Apply`.

## Тесты

- `discounts/round_test.go` — half-to-even.
- `discounts/*_test.go` — позитив + негатив для каждой из 7 стратегий.
- `engine/engine_test.go` — 10 интеграционных: порядок стадий, пара промокод/процент (больше/равенство), нижняя граница, округление, границы (порог, 30 дней, дата промокода, доставка), breakdown, пустая корзина, неизвестный/просроченный промокод, все совместимые скидки.
- `provider/provider_test.go` — тот же указатель, set/get, конкурентный доступ.

## Демо

`cmd/demo/main.go` собирает конфиг из 5 скидок + промокод и печатает `PricingResult` в JSON. Суммы в копейках. Пример вывода:

```json
{
  "BaseTotal": 650000,
  "AppliedDiscounts": [
    { "Name": "3 по цене 2", "Amount": 200000, "Reason": "Категория Книги: 6 единиц, 2 бесплатно" },
    { "Name": "Промокод 15%", "Amount": 60000, "Reason": "Промокод SAVE15: скидка 600.00 руб." },
    { "Name": "Лояльность", "Amount": 17000, "Reason": "Лояльность: последняя покупка 12 дней назад" },
    { "Name": "500 рублей от 3000", "Amount": 50000, "Reason": "Фиксированная скидка 500.00 руб. на товары" }
  ],
  "FinalTotal": 323000
}
```