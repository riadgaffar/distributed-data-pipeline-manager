package parsers

import (
	"testing"
)

func TestJSONParser(t *testing.T) {
	parser := &JSONParser{}

	// Valid JSON input
	validJSON := []byte(`["Message 1", "Message 2", "Message 3"]`)

	messages, err := parser.Parse(validJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// Invalid JSON input
	invalidJSON := []byte(`{"messages": "not an array"}`)
	_, err = parser.Parse(invalidJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
