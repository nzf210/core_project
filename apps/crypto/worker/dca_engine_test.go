package main

import (
	"testing"
	"time"
	"core_project/apps/crypto/domain"
)

func TestShouldRunDCA(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name           string
		lastExecutedAt *time.Time
		interval       string
		expected       bool
	}{
		{
			name:           "Nil lastExecutedAt (First run)",
			lastExecutedAt: nil,
			interval:       domain.DCAIntervalDaily,
			expected:       true,
		},
		{
			name:           "Hourly interval, 30 mins ago (Should not run)",
			lastExecutedAt: ptrTime(now.Add(-30 * time.Minute)),
			interval:       domain.DCAIntervalHourly,
			expected:       false,
		},
		{
			name:           "Hourly interval, 65 mins ago (Should run)",
			lastExecutedAt: ptrTime(now.Add(-65 * time.Minute)),
			interval:       domain.DCAIntervalHourly,
			expected:       true,
		},
		{
			name:           "Daily interval, 12 hours ago (Should not run)",
			lastExecutedAt: ptrTime(now.Add(-12 * time.Hour)),
			interval:       domain.DCAIntervalDaily,
			expected:       false,
		},
		{
			name:           "Daily interval, 25 hours ago (Should run)",
			lastExecutedAt: ptrTime(now.Add(-25 * time.Hour)),
			interval:       domain.DCAIntervalDaily,
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRunDCA(tt.lastExecutedAt, tt.interval)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
