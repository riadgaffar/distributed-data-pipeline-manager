package loader

import (
	"distributed-data-pipeline-manager/src/parsers"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
)

// LoadParser loads a parser plugin or returns the default JSON parser
func LoadParser(pluginPath string, parserType string) (parsers.Parser, error) {
	log.Printf("DEBUG: Loading plugin from path: %s", pluginPath)

	// Check if file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file does not exist at %s", pluginPath)
	}

	// If no plugin path provided, use default plugin path
	if pluginPath == "" {
		defaultPath := filepath.Join("bin", "plugins", parserType+".so")
		pluginPath = defaultPath
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

func getDefaultParser(parserType string) (parsers.Parser, error) {
	switch parserType {
	case "json":
		return &parsers.JSONParser{}, nil
	case "avro", "parquet":
		return nil, fmt.Errorf("parser type '%s' requires plugin implementation", parserType)
	default:
		return nil, fmt.Errorf("unsupported parser type: %s", parserType)
	}
}
