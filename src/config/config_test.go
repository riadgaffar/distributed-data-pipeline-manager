package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// mockAppConfig generates a mock AppConfig for testing
func mockAppConfig() *AppConfig {
	return &AppConfig{
		App: struct {
			Profiling bool `yaml:"profiling"`
			Source    struct {
				Parser string `yaml:"parser"`
				File   string `yaml:"file"`
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
			LoggerConfig struct {
				Level string `yaml:"level"`
			} `yaml:"logger"`
		}{
			Profiling: false,
			Source: struct {
				Parser string `yaml:"parser"`
				File   string `yaml:"file"`
			}{
				Parser: "json",
				File:   "test-messages.json",
			},
			Kafka: struct {
				Brokers       []string `yaml:"brokers"`
				Topics        []string `yaml:"topics"`
				ConsumerGroup string   `yaml:"consumer_group"`
			}{
				Brokers:       []string{"localhost:9092"},
				Topics:        []string{"test-topic"},
				ConsumerGroup: "test-group",
			},
			Postgres: struct {
				URL   string `yaml:"url"`
				Table string `yaml:"table"`
			}{
				URL:   "postgresql://user:password@localhost:5432/test_db?sslmode=disable",
				Table: "test_table",
			},
			LoggerConfig: struct {
				Level string `yaml:"level"`
			}{
				Level: "DEBUG",
			},
		},
	}
}

// writeMockConfigFile writes a mock AppConfig YAML to a temporary file
func writeMockConfigFile(t *testing.T, config *AppConfig) string {
	t.Helper()
	tmpFile := filepath.Join(os.TempDir(), "test_app_config.yaml")
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

func TestLoadConfig_Success(t *testing.T) {
	// Create a mock configuration
	mockConfig := mockAppConfig()

	// Write the mock configuration to a temporary file
	tmpFile := writeMockConfigFile(t, mockConfig)
	defer os.Remove(tmpFile)

	// Set CONFIG_PATH environment variable
	os.Setenv("CONFIG_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_PATH")

	// Load the configuration
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Validate the loaded configuration
	if config.App.Kafka.ConsumerGroup != mockConfig.App.Kafka.ConsumerGroup {
		t.Errorf("expected ConsumerGroup %s, got %s", mockConfig.App.Kafka.ConsumerGroup, config.App.Kafka.ConsumerGroup)
	}
	if config.App.Postgres.Table != mockConfig.App.Postgres.Table {
		t.Errorf("expected Postgres Table %s, got %s", mockConfig.App.Postgres.Table, config.App.Postgres.Table)
	}
	if config.App.LoggerConfig.Level != mockConfig.App.LoggerConfig.Level {
		t.Errorf("expected Logger Level %s, got %s", mockConfig.App.LoggerConfig.Level, config.App.LoggerConfig.Level)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	// Set CONFIG_PATH to a non-existent file
	os.Setenv("CONFIG_PATH", "nonexistent_file.yaml")
	defer os.Unsetenv("CONFIG_PATH")

	_, err := LoadConfig("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !os.IsNotExist(err) && !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("expected file not found error, got %v", err)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML content
	tmpFile := filepath.Join(os.TempDir(), "invalid_config.yaml")
	err := os.WriteFile(tmpFile, []byte("invalid_yaml: [unbalanced"), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid YAML: %v", err)
	}
	defer os.Remove(tmpFile)

	// Set CONFIG_PATH to the invalid file
	os.Setenv("CONFIG_PATH", tmpFile)
	defer os.Unsetenv("CONFIG_PATH")

	_, err = LoadConfig("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check if the error contains the expected message
	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	// Create a mock configuration
	mockConfig := mockAppConfig()
	filePath := writeMockConfigFile(t, mockConfig)
	defer os.Remove(filePath)

	// Set CONFIG_PATH environment variable
	os.Setenv("CONFIG_PATH", filePath)
	defer os.Unsetenv("CONFIG_PATH")

	// Load the configuration
	loadedConfig, err := LoadConfig("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Validate the loaded configuration
	if loadedConfig.App.Kafka.Brokers[0] != mockConfig.App.Kafka.Brokers[0] {
		t.Errorf("expected Kafka Broker %s, got %s", mockConfig.App.Kafka.Brokers[0], loadedConfig.App.Kafka.Brokers[0])
	}
}
