package parsers

// Parser defines a common interface for all parsers.
type Parser interface {
	Parse(data []byte) (interface{}, error) // Parses input data
	Name() string                           // Returns the parser name
	Version() string                        // Returns the parser version
}
