package main

import (
	"fmt"
	"sort"
)

const (
	CycleMonthly = "monthly"
	CycleYearly  = "yearly"
	Cycle2Yearly = "2yearly"
)

var cycleDivisors = map[string]float64{
	CycleMonthly: 1,
	CycleYearly:  12,
	Cycle2Yearly: 24,
}

type Subscription struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Cycle     string  `json:"cycle"`
	StartDate string  `json:"start_date"`
	Notes     string  `json:"notes"`
}

type Summary struct {
	Totals    []CurrencyTotal    `json:"totals"`
	Breakdown []SubscriptionLine `json:"breakdown"`
}

type CurrencyTotal struct {
	Currency string  `json:"currency"`
	Monthly  float64 `json:"monthly"`
	Yearly   float64 `json:"yearly"`
}

type SubscriptionLine struct {
	Name              string  `json:"name"`
	Currency          string  `json:"currency"`
	MonthlyEquivalent float64 `json:"monthly_equivalent"`
}

func IsValidCycle(cycle string) bool {
	_, ok := cycleDivisors[cycle]
	return ok
}

func MonthlyEquivalent(amount float64, cycle string) (float64, error) {
	divisor, ok := cycleDivisors[cycle]
	if !ok {
		return 0, fmt.Errorf("invalid cycle %q", cycle)
	}

	return amount / divisor, nil
}

func BuildSummary(subscriptions []Subscription) (Summary, error) {
	summary := Summary{
		Totals:    []CurrencyTotal{},
		Breakdown: []SubscriptionLine{},
	}
	totalsByCurrency := make(map[string]float64)

	for _, sub := range subscriptions {
		monthly, err := MonthlyEquivalent(sub.Amount, sub.Cycle)
		if err != nil {
			return Summary{}, err
		}

		summary.Breakdown = append(summary.Breakdown, SubscriptionLine{
			Name:              sub.Name,
			Currency:          sub.Currency,
			MonthlyEquivalent: monthly,
		})
		totalsByCurrency[sub.Currency] += monthly
	}

	currencies := make([]string, 0, len(totalsByCurrency))
	for currency := range totalsByCurrency {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)

	for _, currency := range currencies {
		monthly := totalsByCurrency[currency]
		summary.Totals = append(summary.Totals, CurrencyTotal{
			Currency: currency,
			Monthly:  monthly,
			Yearly:   monthly * 12,
		})
	}

	return summary, nil
}
