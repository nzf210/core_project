package handlers

import (
	"testing"
)

func TestSplitExternalID(t *testing.T) {
	parts := splitExternalID("INV-abc123|tenant-456")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[0] != "INV-abc123" {
		t.Errorf("got %q, want INV-abc123", parts[0])
	}
	if parts[1] != "tenant-456" {
		t.Errorf("got %q, want tenant-456", parts[1])
	}
}

func TestSplitExternalID_NoSeparator(t *testing.T) {
	parts := splitExternalID("INV-abc123")
	if len(parts) != 1 {
		t.Errorf("expected 1 part, got %d", len(parts))
	}
}

func TestCalculatePrice_WargameToken(t *testing.T) {
	price, ok := calculatePrice("wargame_token", 3)
	if !ok {
		t.Error("expected ok=true for wargame_token")
	}
	if price != 300_000 {
		t.Errorf("expected 300000, got %d", price)
	}
}

func TestCalculatePrice_IntelligencePack(t *testing.T) {
	price, ok := calculatePrice("intelligence_pack", 2)
	if !ok {
		t.Error("expected ok=true for intelligence_pack")
	}
	if price != 10_000_000 {
		t.Errorf("expected 10000000, got %d", price)
	}
}

func TestCalculatePrice_Unknown(t *testing.T) {
	price, ok := calculatePrice("unknown_type", 1)
	if ok {
		t.Error("expected ok=false for unknown type")
	}
	if price != 0 {
		t.Errorf("expected price=0, got %d", price)
	}
}

func TestParseAgeRange_Empty(t *testing.T) {
	min, max, err := parseAgeRange("")
	if err != nil || min != 0 || max != 0 {
		t.Errorf("expected 0,0,nil for empty, got %d,%d,%v", min, max, err)
	}
}

func TestParseAgeRange_Range(t *testing.T) {
	min, max, err := parseAgeRange("18-25")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if min != 18 || max != 25 {
		t.Errorf("expected 18-25, got %d-%d", min, max)
	}
}

func TestParseAgeRange_Plus(t *testing.T) {
	min, max, err := parseAgeRange("60+")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if min != 60 || max != 0 {
		t.Errorf("expected min=60 max=0, got %d,%d", min, max)
	}
}

func TestParseAgeRange_InvalidRange(t *testing.T) {
	_, _, err := parseAgeRange("25-18") // max < min
	if err == nil {
		t.Error("expected error for invalid range")
	}
}

func TestParseAgeRange_InvalidFormat(t *testing.T) {
	_, _, err := parseAgeRange("young")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}
