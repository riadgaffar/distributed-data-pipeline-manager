package parsers

import (
	"testing"
)

func TestJSONParser(t *testing.T) {
	parser := &JSONParser{}

	t.Run("verify parser name", func(t *testing.T) {
		if name := parser.Name(); name != "json" {
			t.Errorf("expected name 'json', got %s", name)
		}
	})

	t.Run("verify parser version", func(t *testing.T) {
		if version := parser.Version(); version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %s", version)
		}
	})

	t.Run("parse valid JSON", func(t *testing.T) {
		input := []byte(`{"key": "value"}`)
		result, err := parser.Parse(input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("expected parsed result, got nil")
		}
	})
}
