package main

import (
	"encoding/json"
	"fmt"
)

type JSONParser struct{}

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

func (p *JSONParser) Name() string {
	return "json"
}

func (p *JSONParser) Version() string {
	return "1.0.0"
}

// Export the parser
var Parser JSONParser
