package domain

import (
	"testing"
)

func TestUSDTConversions(t *testing.T) {
	tests := []struct {
		name      string
		usdtFloat float64
		cents     int64
	}{
		{"10 USDT", 10.0, 1000},
		{"0.5 USDT", 0.5, 50},
		{"100.25 USDT", 100.25, 10025},
		{"0 USDT", 0.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCents := USDTToCents(tt.usdtFloat)
			if gotCents != tt.cents {
				t.Errorf("USDTToCents(%f) = %d, want %d", tt.usdtFloat, gotCents, tt.cents)
			}

			gotUSDT := CentsToUSDT(tt.cents)
			if gotUSDT != tt.usdtFloat {
				t.Errorf("CentsToUSDT(%d) = %f, want %f", tt.cents, gotUSDT, tt.usdtFloat)
			}
		})
	}
}
