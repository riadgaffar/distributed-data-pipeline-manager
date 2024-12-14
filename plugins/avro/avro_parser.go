package main

import (
	"encoding/json"
	"fmt"

	"github.com/linkedin/goavro"
)

// AvroParser implements the parsers.Parser interface for Avro format
type AvroParser struct {
	schema string // Avro schema for serialization and deserialization
}

// NewAvroParser creates a new instance of AvroParser with the provided schema
func NewAvroParser(schema string) (*AvroParser, error) {
	if !IsValidAvroSchema(schema) {
		return nil, fmt.Errorf("invalid Avro schema")
	}
	return &AvroParser{schema: schema}, nil
}

// IsValidAvroSchema validates if the provided schema is a valid Avro schema
func IsValidAvroSchema(schema string) bool {
	_, err := goavro.NewCodec(schema)
	return err == nil
}

// Parse method with dynamic validation
func (p *AvroParser) Parse(data []byte) (interface{}, error) {
	codec, err := goavro.NewCodec(p.schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create Avro codec: %w", err)
	}

	nativeData, _, err := codec.NativeFromBinary(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize Avro data: %w", err)
	}

	// Validate the deserialized data structure
	nativeMap, ok := nativeData.(map[string]interface{})
	if !ok || len(nativeMap) == 0 {
		return nil, fmt.Errorf("unexpected or empty data format: %T", nativeData)
	}

	// Dynamic validation based on schema
	if err := p.validateDynamicFields(nativeMap); err != nil {
		return nil, err
	}

	return nativeMap, nil
}

// validateDynamicFields dynamically validates fields based on the schema
func (p *AvroParser) validateDynamicFields(data map[string]interface{}) error {
	var schemaMap map[string]interface{}

	// Parse schema JSON into a map
	if err := json.Unmarshal([]byte(p.schema), &schemaMap); err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Extract field definitions
	fields, ok := schemaMap["fields"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid schema structure: missing fields")
	}

	// Validate required fields
	for _, field := range fields {
		fieldMap, ok := field.(map[string]interface{})
		if !ok {
			continue
		}

		fieldName, _ := fieldMap["name"].(string)
		fieldType := fieldMap["type"]

		// Check if the field exists in the data
		if value, exists := data[fieldName]; !exists || isFieldEmpty(value, fieldType) {
			return fmt.Errorf("field '%s' is missing or invalid", fieldName)
		}
	}

	return nil
}

// isFieldEmpty checks if a field value is empty based on its type
func isFieldEmpty(value interface{}, fieldType interface{}) bool {
	switch expectedType := fieldType.(type) {
	case string: // Primitive types like "string", "int", etc.
		switch expectedType {
		case "string":
			if v, ok := value.(string); ok {
				return v == "" // Empty string is considered empty
			}
		case "int", "long", "float", "double":
			return value == nil // Check for nil numbers
		case "boolean":
			if _, ok := value.(bool); ok {
				return false // Booleans are never empty if set
			}
			return true
		case "null":
			return value == nil // Explicitly check for null values
		default:
			return value == nil // Default: treat nil as empty
		}

	case []interface{}: // Union types like ["null", "string"]
		// Validate against each possible type in the union
		for _, unionType := range expectedType {
			if !isFieldEmpty(value, unionType) {
				return false // At least one type matches
			}
		}
		return true

	case map[string]interface{}: // Complex types like "record", "map"
		// Implement additional logic for records, maps, etc.
		if expectedType["type"] == "record" {
			return value == nil // Example: Record must not be nil
		}

	default:
		return value == nil // Fallback for unsupported types
	}

	return false
}

// Serialize serializes a Go map into Avro binary format using the parser's schema
func (p *AvroParser) Serialize(data map[string]interface{}) ([]byte, error) {
	codec, err := goavro.NewCodec(p.schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create Avro codec: %w", err)
	}

	binaryData, err := codec.BinaryFromNative(nil, data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data to Avro: %w", err)
	}

	return binaryData, nil
}

// Name returns the name of the parser
func (p *AvroParser) Name() string {
	return "avro"
}

// Version returns the version of the parser
func (p *AvroParser) Version() string {
	return "1.0.0"
}
