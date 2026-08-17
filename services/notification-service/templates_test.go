package main

import (
	"testing"
)

func TestRenderTemplate_StringValue(t *testing.T) {
	result := RenderTemplate("Hello {{name}}!", map[string]interface{}{"name": "World"})
	if result != "Hello World!" {
		t.Errorf("got %q, want %q", result, "Hello World!")
	}
}

func TestRenderTemplate_Float64Value(t *testing.T) {
	result := RenderTemplate("Total: {{amount}}", map[string]interface{}{"amount": float64(99.5)})
	if result != "Total: 99.5" {
		t.Errorf("got %q, want %q", result, "Total: 99.5")
	}
}

func TestRenderTemplate_DefaultValue(t *testing.T) {
	result := RenderTemplate("Count: {{n}}", map[string]interface{}{"n": 42})
	if result != "Count: 42" {
		t.Errorf("got %q, want %q", result, "Count: 42")
	}
}

func TestRenderTemplate_MultipleKeys(t *testing.T) {
	result := RenderTemplate("{{greeting}} {{name}}, your order {{id}} is ready.", map[string]interface{}{
		"greeting": "Halo",
		"name":     "Budi",
		"id":       "ORD-001",
	})
	if result != "Halo Budi, your order ORD-001 is ready." {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestRenderTemplate_MissingKeyPreservesPlaceholder(t *testing.T) {
	result := RenderTemplate("Hello {{name}} and {{other}}!", map[string]interface{}{"name": "Budi"})
	if result != "Hello Budi and {{other}}!" {
		t.Errorf("expected partial substitution, got %q", result)
	}
}

func TestRenderTemplate_EmptyTemplateWithData(t *testing.T) {
	result := RenderTemplate("", map[string]interface{}{"key": "val"})
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestRenderTemplate_NilData(t *testing.T) {
	result := RenderTemplate("Hello {{name}}!", nil)
	if result != "Hello {{name}}!" {
		t.Errorf("expected unchanged template with nil data, got %q", result)
	}
}
