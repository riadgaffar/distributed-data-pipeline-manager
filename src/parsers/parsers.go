package parsers

// Parser defines a common interface for all parsers.
type Parser interface {
	Parse(data []byte) ([]string, error) // Parses input data and returns a list of messages
}
