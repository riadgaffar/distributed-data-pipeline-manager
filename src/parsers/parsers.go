package parsers

import "fmt"

// Parser defines a common interface for all parsers.
type Parser interface {
	Initialize(schema string) (interface{}, error)              // Initializes the parser with a schema
	Parse(data []byte) (interface{}, error)                     // Parses input data
	Serialize(data map[string]interface{}) (interface{}, error) // Serializes input data for parsers that support serialization
	Name() string                                               // Returns the parser name
	Version() string                                            // Returns the parser version
}

// Register parsers (JSON, Avro, etc.)
var parserRegistry = map[string]Parser{}

// RegisterParser adds a parser to the registry
func RegisterParser(format string, parser Parser) {
	parserRegistry[format] = parser
}

// ParsePayload routes the payload to the appropriate parser based on format
func ParsePayload(payload map[string]interface{}) (interface{}, error) {
	format, ok := payload["format"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid format field")
	}

	data, ok := payload["data"].(string) // Assuming the `data` is serialized
	if !ok {
		return nil, fmt.Errorf("missing or invalid data field")
	}

	// Fetch parser from registry
	parser, exists := parserRegistry[format]
	if !exists {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Debug log
	fmt.Printf("Received data to parse: %v\n", data)

	// Use the parser to process the data
	return parser.Parse([]byte(data))
}
