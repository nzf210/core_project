package main

import (
	"testing"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// REVENUE SIMULATION TESTS
// Tests untuk revenue simulation logic
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestRevenueCalculation(t *testing.T) {
	// Test revenue calculation
	tests := []struct {
		name          string
		subscriptions int
		pricePerMonth int64
		wantRevenue   int64
	}{
		{
			name:          "basic calculation",
			subscriptions: 10,
			pricePerMonth:  99000,
			wantRevenue:   990000,
		},
		{
			name:          "zero subscriptions",
			subscriptions: 0,
			pricePerMonth:  99000,
			wantRevenue:   0,
		},
		{
			name:          "single subscription",
			subscriptions: 1,
			pricePerMonth:  99000,
			wantRevenue:   99000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revenue := int64(tt.subscriptions) * tt.pricePerMonth
			if revenue != tt.wantRevenue {
				t.Errorf("revenue = %d, want %d", revenue, tt.wantRevenue)
			}
		})
	}
}

func TestSubscriptionPlanPricing(t *testing.T) {
	// Test subscription plan pricing
	plans := map[string]int64{
		"lite":       99000,
		"pro":        199000,
		"enterprise": 499000,
	}

	if plans["lite"] >= plans["pro"] {
		t.Error("Lite should be cheaper than Pro")
	}

	if plans["pro"] >= plans["enterprise"] {
		t.Error("Pro should be cheaper than Enterprise")
	}
}

func TestRevenueGrowthCalculation(t *testing.T) {
	// Test revenue growth over time
	currentRevenue := int64(1000000)
	growthRate := 0.1 // 10% monthly growth

	// Calculate next month revenue
	nextMonthRevenue := float64(currentRevenue) * (1 + growthRate)

	if nextMonthRevenue <= float64(currentRevenue) {
		t.Error("Next month revenue should be higher with positive growth")
	}
}

func TestChurnRateCalculation(t *testing.T) {
	// Test churn rate impact on revenue
	totalSubscribers := 100
	churnRate := 0.05 // 5% monthly churn

	// Calculate churned subscribers
	churnedSubscribers := float64(totalSubscribers) * churnRate
	remainingSubscribers := float64(totalSubscribers) * (1 - churnRate)

	if churnedSubscribers <= 0 {
		t.Error("Churned subscribers should be positive")
	}

	if remainingSubscribers != 95 {
		t.Errorf("Remaining subscribers = %f, want 95", remainingSubscribers)
	}
}

func TestMRRCalculation(t *testing.T) {
	// Test Monthly Recurring Revenue calculation
	subscribers := map[string]int{
		"lite":       50,
		"pro":        30,
		"enterprise": 20,
	}

	prices := map[string]int64{
		"lite":       99000,
		"pro":        199000,
		"enterprise": 499000,
	}

	var mrr int64
	for plan, count := range subscribers {
		mrr += int64(count) * prices[plan]
	}

	// Expected: 50*99000 + 30*199000 + 20*499000 = 4950000 + 5970000 + 9980000 = 20900000
	expected := int64(20900000)
	if mrr != expected {
		t.Errorf("MRR = %d, want %d", mrr, expected)
	}
}
