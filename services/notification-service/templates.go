package main

import (
	"fmt"
	"strings"
)

// RenderTemplate replaces {{key}} with value from data map
func RenderTemplate(template string, data map[string]interface{}) string {
	result := template
	for k, v := range data {
		placeholder := "{{" + k + "}}"
		strVal := ""
		switch val := v.(type) {
		case string:
			strVal = val
		case float64: // JSON numbers are float64 by default
			strVal = fmt.Sprintf("%v", val)
		default:
			strVal = fmt.Sprintf("%v", val)
		}
		result = strings.ReplaceAll(result, placeholder, strVal)
	}
	return result
}
