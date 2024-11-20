package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// mockPipelineConfig generates a mock pipeline configuration for testing
func mockPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		Input: InputConfig{
			Kafka: KafkaConfig{
				Addresses:     []string{"broker1:9092", "broker2:9092"},
				Topics:        []string{"topic1", "topic2"},
				ConsumerGroup: "test-group",
			},
		},
		Processors: []Processor{
			{Type: "bloblang", Bloblang: `root = this`},
		},
		Output: OutputConfig{
			Type: "postgres",
			Postgres: PostgresConfig{
				URL:   "postgresql://user:password@localhost:5432/db",
				Table: "output_table",
				Columns: map[string]string{
					"field1": "column1",
					"field2": "column2",
				},
			},
		},
		Logger: LoggerConfig{
			Level: "info",
		},
	}
}

// writeMockConfigFile writes a mock YAML configuration to a temporary file
func writeMockConfigFile(t *testing.T, config *PipelineConfig) string {
	t.Helper()
	tmpFile := filepath.Join(os.TempDir(), "test_config.yaml")
	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		t.Fatalf("failed to write mock config: %v", err)
	}

	return tmpFile
}

func TestParsePipelineConfig_Success(t *testing.T) {
	// Create a mock configuration
	mockConfig := mockPipelineConfig()
	filePath := writeMockConfigFile(t, mockConfig)
	defer os.Remove(filePath)

	// Parse the configuration
	parsedConfig, err := ParsePipelineConfig(filePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Validate the parsed configuration
	if parsedConfig.Input.Kafka.ConsumerGroup != mockConfig.Input.Kafka.ConsumerGroup {
		t.Errorf("expected ConsumerGroup %s, got %s", mockConfig.Input.Kafka.ConsumerGroup, parsedConfig.Input.Kafka.ConsumerGroup)
	}
	if parsedConfig.Output.Type != mockConfig.Output.Type {
		t.Errorf("expected Output Type %s, got %s", mockConfig.Output.Type, parsedConfig.Output.Type)
	}
	if parsedConfig.Logger.Level != mockConfig.Logger.Level {
		t.Errorf("expected Logger Level %s, got %s", mockConfig.Logger.Level, parsedConfig.Logger.Level)
	}
}

func TestParsePipelineConfig_FileNotFound(t *testing.T) {
	_, err := ParsePipelineConfig("nonexistent_file.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !os.IsNotExist(err) && !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("expected file not found error, got %v", err)
	}
}

func TestParsePipelineConfig_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML content
	tmpFile := filepath.Join(os.TempDir(), "invalid_config.yaml")
	err := os.WriteFile(tmpFile, []byte("invalid_yaml: [unbalanced"), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid YAML: %v", err)
	}
	defer os.Remove(tmpFile)

	// Attempt to parse the configuration
	_, err = ParsePipelineConfig(tmpFile)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check if the error contains the expected message
	if !strings.Contains(err.Error(), "failed to parse configuration") {
		t.Errorf("unexpected error message: %v", err)
	}
}
