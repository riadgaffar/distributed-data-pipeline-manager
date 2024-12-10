package loader

import (
	"distributed-data-pipeline-manager/src/parsers"
	"fmt"
	"log"
	"os"
	"plugin"
)

// LoadParser loads a parser plugin or returns the default JSON parser
func LoadParser(pluginPath string, parserType string) (parsers.Parser, error) {
	log.Printf("DEBUG: Loading plugin from path: %s", pluginPath)

	// Check if file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file does not exist at %s", pluginPath)
	}

	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %s (path: %s): %w", parserType, pluginPath, err)
	}

	symParser, err := p.Lookup("Parser")
	if err != nil {
		return nil, fmt.Errorf("plugin %s doesn't implement Parser: %w", parserType, err)
	}

	parser, ok := symParser.(parsers.Parser)
	if !ok {
		return nil, fmt.Errorf("invalid parser plugin type for %s: %T", parserType, symParser)
	}

	return parser, nil
}
