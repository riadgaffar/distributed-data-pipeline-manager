package parsers

import (
	"encoding/json"
	"fmt"
)

// JSONParser implements the Parser interface for JSON data.
type JSONParser struct{}

// JSONPayload represents the expected structure of the JSON input.
type JSONPayload struct {
	Messages []string `json:"messages"`
}

// Parse parses the JSON data and returns a list of messages.
func (j *JSONParser) Parse(data []byte) ([]string, error) {
	var messages []string
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return messages, nil
}
