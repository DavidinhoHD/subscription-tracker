package main

import (
	"math"
	"testing"
)

func TestMonthlyEquivalent(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		cycle   string
		want    float64
		wantErr bool
	}{
		{name: "monthly", amount: 9.99, cycle: CycleMonthly, want: 9.99},
		{name: "yearly", amount: 120, cycle: CycleYearly, want: 10},
		{name: "2yearly", amount: 240, cycle: Cycle2Yearly, want: 10},
		{name: "invalid", amount: 10, cycle: "weekly", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MonthlyEquivalent(tt.amount, tt.cycle)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !floatEqual(got, tt.want) {
				t.Fatalf("MonthlyEquivalent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSummary(t *testing.T) {
	subscriptions := []Subscription{
		{Name: "Netflix", Amount: 9.99, Currency: "EUR", Cycle: CycleMonthly},
		{Name: "Adobe CC", Amount: 75, Currency: "EUR", Cycle: CycleYearly},
		{Name: "Domain", Amount: 48, Currency: "USD", Cycle: Cycle2Yearly},
	}

	got, err := BuildSummary(subscriptions)
	if err != nil {
		t.Fatalf("BuildSummary() error = %v", err)
	}

	if len(got.Breakdown) != 3 {
		t.Fatalf("len(Breakdown) = %d, want 3", len(got.Breakdown))
	}
	if got.Breakdown[0].Name != "Netflix" || !floatEqual(got.Breakdown[0].MonthlyEquivalent, 9.99) {
		t.Fatalf("unexpected first breakdown item: %#v", got.Breakdown[0])
	}
	if got.Breakdown[1].Name != "Adobe CC" || !floatEqual(got.Breakdown[1].MonthlyEquivalent, 6.25) {
		t.Fatalf("unexpected second breakdown item: %#v", got.Breakdown[1])
	}

	if len(got.Totals) != 2 {
		t.Fatalf("len(Totals) = %d, want 2", len(got.Totals))
	}
	if got.Totals[0].Currency != "EUR" {
		t.Fatalf("first currency = %q, want EUR", got.Totals[0].Currency)
	}
	if !floatEqual(got.Totals[0].Monthly, 16.24) || !floatEqual(got.Totals[0].Yearly, 194.88) {
		t.Fatalf("unexpected EUR totals: %#v", got.Totals[0])
	}
	if got.Totals[1].Currency != "USD" {
		t.Fatalf("second currency = %q, want USD", got.Totals[1].Currency)
	}
	if !floatEqual(got.Totals[1].Monthly, 2) || !floatEqual(got.Totals[1].Yearly, 24) {
		t.Fatalf("unexpected USD totals: %#v", got.Totals[1])
	}
}

func TestBuildSummaryEmpty(t *testing.T) {
	got, err := BuildSummary(nil)
	if err != nil {
		t.Fatalf("BuildSummary() error = %v", err)
	}
	if got.Totals == nil {
		t.Fatal("Totals is nil, want empty slice")
	}
	if got.Breakdown == nil {
		t.Fatal("Breakdown is nil, want empty slice")
	}
	if len(got.Totals) != 0 || len(got.Breakdown) != 0 {
		t.Fatalf("BuildSummary() = %#v, want empty totals and breakdown", got)
	}
}

func TestBuildSummaryInvalidCycle(t *testing.T) {
	_, err := BuildSummary([]Subscription{{Name: "Bad", Amount: 1, Currency: "EUR", Cycle: "weekly"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.000001
}
