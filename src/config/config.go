package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RealConfigLoader struct{}

// AppConfig defines the structure for dynamic configuration
type AppConfig struct {
	App struct {
		Profiling              bool   `yaml:"profiling"`
		PipelineTemplate       string `yaml:"pipeline_template"`
		GeneratePipelineConfig string `yaml:"generated_pipeline_config"`
		Source                 struct {
			Parser     string `yaml:"parser"`
			PluginPath string `yaml:"plugin_path"`
		} `yaml:"source"`
		Kafka struct {
			Brokers       []string `yaml:"brokers"`
			Topics        []string `yaml:"topics"`
			ConsumerGroup string   `yaml:"consumer_group"`
		} `yaml:"kafka"`
		Postgres struct {
			URL   string `yaml:"url"`
			Table string `yaml:"table"`
		} `yaml:"postgres"`
		Logger struct {
			Level string `yaml:"level"`
		} `yaml:"logger"`
	} `yaml:"app"`
}

// LoadConfig loads the YAML configuration file.
// It first checks for the CONFIG_PATH environment variable.
// If a specific path is provided as an argument, it overrides the environment variable.
func LoadConfig(explicitPath ...string) (*AppConfig, error) {
	var configPath string

	// If an explicit path is provided, use it; otherwise, check CONFIG_PATH
	if len(explicitPath) > 0 && explicitPath[0] != "" {
		configPath = explicitPath[0]
	} else {
		configPath = os.Getenv("CONFIG_PATH")
		if configPath == "" {
			return nil, fmt.Errorf("CONFIG_PATH environment variable is not set, and no explicit path provided")
		}
	}

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", configPath, err)
	}

	// Parse the YAML configuration into AppConfig
	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func NewConfigLoader() *RealConfigLoader {
	return &RealConfigLoader{}
}

func (l *RealConfigLoader) LoadConfig(path string) (*AppConfig, error) {
	return LoadConfig(path)
}
