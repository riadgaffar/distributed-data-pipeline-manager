package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAvroSchemaValidation(t *testing.T) {
	validSchema := `{
		"type": "record",
		"name": "TestRecord",
		"fields": [{"name": "field1", "type": "string"}]
	}`
	invalidSchema := `{
		"type": "record",
		"name": "TestRecord",
		"fields": [{"name": "field1", "type": "unknown_type"}]
	}`

	assert.True(t, IsValidAvroSchema(validSchema), "Expected schema to be valid")
	assert.False(t, IsValidAvroSchema(invalidSchema), "Expected schema to be invalid")
}

func TestAvroSerializationAndDeserialization(t *testing.T) {
	schema := `{
		"type": "record",
		"name": "TestRecord",
		"fields": [{"name": "field1", "type": "string"}]
	}`
	parser, err := NewAvroParser(schema)
	assert.NoError(t, err, "Parser creation should not return an error")

	// Test data
	data := map[string]interface{}{"field1": "test_value"}

	// Serialize
	encodedData, err := parser.Serialize(data)
	assert.NoError(t, err, "Serialization should not return an error")
	assert.NotEmpty(t, encodedData, "Encoded data should not be empty")

	// Deserialize
	decodedData, err := parser.Parse(encodedData)
	assert.NoError(t, err, "Deserialization should not return an error")
	assert.Equal(t, data["field1"], decodedData.(map[string]interface{})["field1"], "Decoded value should match original")
}

func TestAvroParserInvalidBinary(t *testing.T) {
	schema := `{
			"type": "record",
			"name": "TestRecord",
			"fields": [{"name": "field1", "type": "string"}]
	}`
	parser, err := NewAvroParser(schema)
	assert.NoError(t, err, "Parser creation should not return an error")

	invalidBinary := []byte{0x00, 0x01, 0x02}

	_, err = parser.Parse(invalidBinary)
	assert.Error(t, err, "Parsing invalid binary data should return an error")
}
