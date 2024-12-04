package parsers

import (
	"encoding/json"
	"fmt"
)

// JSONParser implements the Parser interface for JSON data.
type JSONParser struct{}

// Parse parses the JSON data and returns a list of messages.
func (p *JSONParser) Parse(data []byte) (interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("failed to parse JSON: unexpected end of JSON input")
	}

	var payload interface{}
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return payload, nil
}
