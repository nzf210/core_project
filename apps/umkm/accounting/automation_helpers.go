package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)


func getAutomationLimit(plan string) int {
	switch plan {
	case "lite":
		return 3
	case "pro":
		return 10
	case "enterprise", "ultimate", "superadmin":
		return 999
	default:
		return 0 // unknown / inactive = no automation
	}
}

func cronMatchesNow(cronExpr string, now time.Time) bool {
	parts := strings.Fields(cronExpr)
	if len(parts) < 5 {
		return false
	}
	minute := now.Minute()
	hour := now.Hour()
	dayOfMonth := now.Day()
	month := int(now.Month())
	dayOfWeek := int(now.Weekday()) // 0=Sunday

	return fieldMatches(parts[0], minute) &&
		fieldMatches(parts[1], hour) &&
		fieldMatches(parts[2], dayOfMonth) &&
		fieldMatches(parts[3], month) &&
		fieldMatches(parts[4], dayOfWeek)
}

func fieldMatches(field string, value int) bool {
	if field == "*" {
		return true
	}
	// Handle */N step values
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step == 0 {
			return false
		}
		return value%step == 0
	}
	// Handle comma-separated values
	for _, part := range strings.Split(field, ",") {
		// Handle range N-M
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				low, err1 := strconv.Atoi(rangeParts[0])
				high, err2 := strconv.Atoi(rangeParts[1])
				if err1 == nil && err2 == nil && value >= low && value <= high {
					return true
				}
			}
			continue
		}
		v, err := strconv.Atoi(part)
		if err == nil && v == value {
			return true
		}
	}
	return false
}

func formatRupiah(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}