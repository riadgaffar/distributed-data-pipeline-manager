package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// KafkaConfig represents the overall structure of the kafka configuration
type KafkaConfig struct {
	Addresses     []string `yaml:"addresses"`
	Topics        []string `yaml:"topics"`
	ConsumerGroup string   `yaml:"consumer_group"`
}

// InputConfig represents the input section of the configuration
type InputConfig struct {
	Kafka KafkaConfig `yaml:"kafka"` // Change Type to KafkaConfig
}

// PipelineConfig represents the overall structure of the pipeline configuration
type PipelineConfig struct {
	Input      InputConfig  `yaml:"input"`
	Processors []Processor  `yaml:"processors"`
	Output     OutputConfig `yaml:"output"`
	Logger     LoggerConfig `yaml:"logger"`
}

// Processor represents each processor in the pipeline
type Processor struct {
	Type     string `yaml:"type"`
	Bloblang string `yaml:"bloblang,omitempty"`
}

// OutputConfig represents the output section of the configuration
type OutputConfig struct {
	Type     string         `yaml:"type"`
	Postgres PostgresConfig `yaml:"postgres"`
}

// PostgresConfig represents the details of the Postgres output configuration
type PostgresConfig struct {
	URL     string            `yaml:"url"`
	Table   string            `yaml:"table"`
	Columns map[string]string `yaml:"columns"`
}

// LoggerConfig represents logging configuration
type LoggerConfig struct {
	Level string `yaml:"level"`
}

// ParsePipelineConfig parses the YAML pipeline configuration into a PipelineConfig struct
func ParsePipelineConfig(filePath string) (*PipelineConfig, error) {
	file, err := os.Open(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	var config PipelineConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return &config, nil
}
