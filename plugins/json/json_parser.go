package main

import (
	"encoding/json"
	"fmt"
)

type JSONParser struct{}

// Initialize the parser with a schema, if needed, NOOP for JSON
func (p *JSONParser) Initialize(schema string) (interface{}, error) {
	return nil, fmt.Errorf("schema initialization is not supported for JSON")
}

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

// Serialize the data, NOOP for JSON
func (p *JSONParser) Serialize(data map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("serialization is not supported for JSON")
}

func (p *JSONParser) Name() string {
	return "json"
}

func (p *JSONParser) Version() string {
	return "1.0.0"
}

// Export the parser
var Parser JSONParser
