package parsers

// Parser defines a common interface for all parsers.
type Parser interface {
	Parse(data []byte) (interface{}, error) // Parses any JSON payload and returns interface{}
}
