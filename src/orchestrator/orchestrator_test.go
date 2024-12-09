package orchestrator

import (
	"distributed-data-pipeline-manager/src/config"
	"os"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockConfigLoader struct {
	mock.Mock
}

// Mock Command Executor
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockConfigLoader) LoadConfig(configPath string) (*config.AppConfig, error) {
	args := m.Called(configPath)
	return args.Get(0).(*config.AppConfig), args.Error(1)
}

// Update ExecuteCommand to use mock.Called
func (m *MockCommandExecutor) ExecuteCommand(name string, args ...string) error {
	callArgs := []interface{}{name}
	for _, arg := range args {
		callArgs = append(callArgs, arg)
	}
	return m.Called(callArgs...).Error(0)
}

// Update StopPipeline to use mock.Called
func (m *MockCommandExecutor) StopPipeline() error {
	return m.Called().Error(0)
}

// Mock Producer
type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	args := m.Called(msg, deliveryChan)
	return args.Error(0)
}

func (m *MockKafkaProducer) Close() {
	m.Called()
}

func (m *MockKafkaProducer) Flush(timeoutMs int) int {
	args := m.Called(timeoutMs)
	return args.Int(0)
}

func createTestConfig() *config.AppConfig {
	return &config.AppConfig{
		App: struct {
			Profiling              bool   `yaml:"profiling"`
			PipelineTemplate       string `yaml:"pipeline_template"`
			GeneratePipelineConfig string `yaml:"generated_pipeline_config"`
			Source                 struct {
				Parser string `yaml:"parser"`
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
		}{
			Profiling:              false,
			PipelineTemplate:       "template",
			GeneratePipelineConfig: "config",
			Source: struct {
				Parser string `yaml:"parser"`
			}{
				Parser: "json",
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
				URL:   "postgres://localhost:5432",
				Table: "table",
			},
			Logger: struct {
				Level string `yaml:"level"`
			}{
				Level: "info",
			},
		},
	}
}

func getCurrentDir() string {
	dir, _ := os.Getwd()
	return dir
}

func setupTestEnv(t *testing.T, configPath, templatePath string) {
	// Create necessary directories
	err := os.MkdirAll("testdata", 0755)
	assert.NoError(t, err, "Failed to create testdata directory")
	err = os.MkdirAll("pipelines/benthos", 0755)
	assert.NoError(t, err, "Failed to create pipelines/benthos directory")

	// Write test configuration file
	testConfig := `
app:
  profiling: false
  pipeline_template: pipelines/benthos/pipeline.yaml
  generated_pipeline_config: pipelines/benthos/generated-pipeline.yaml
  kafka:
    brokers: ["localhost:9092"]
    topics: ["test-topic"]
    consumer_group: "test-group"
  postgres:
    url: "postgresql://user:pass@localhost:5432/db"
    table: "test_table"
  logger:
    level: "DEBUG"
`
	err = os.WriteFile(configPath, []byte(testConfig), 0644)
	assert.NoError(t, err, "Failed to write test configuration file")

	// Write test pipeline template file
	templateContent := `
input:
  kafka:
    addresses: ["${KAFKA_BROKERS}"]
    topics: ["${KAFKA_TOPICS}"]

output:
  database:
    driver: postgres
    dsn: "${POSTGRES_URL}"
    table: "${POSTGRES_TABLE}"
`
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	assert.NoError(t, err, "Failed to write pipeline template file")
}

func cleanupTestEnv(t *testing.T) {
	err := os.RemoveAll("testdata")
	assert.NoError(t, err, "Failed to remove testdata directory")

	err = os.RemoveAll("pipelines")
	assert.NoError(t, err, "Failed to remove pipelines directory")
}

func TestOrchestrator_Run(t *testing.T) {
	// Add debug logging
	t.Logf("Working directory: %s", getCurrentDir())

	configPath := "testdata/config.yaml"
	templatePath := "pipelines/benthos/pipeline.yaml"

	// Setup test environment (directories and files)
	setupTestEnv(t, configPath, templatePath)
	defer cleanupTestEnv(t) // Cleanup after test

	t.Setenv("CONFIG_PATH", "../../config/app-config.yaml")

	// Create mock config
	mockConfig := createTestConfig()

	tests := []struct {
		name          string
		timeout       time.Duration
		setupMocks    func(*MockCommandExecutor, *MockKafkaProducer)
		expectedError error
	}{
		{
			name:    "successful pipeline execution",
			timeout: 2 * time.Second,
			setupMocks: func(executor *MockCommandExecutor, producer *MockKafkaProducer) {
				executor.On("ExecuteCommand", "rpk", "connect", "run", mock.AnythingOfType("string")).Return(nil)
				executor.On("StopPipeline").Return(nil)
				producer.On("Close").Return()
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := new(MockCommandExecutor)
			mockProducer := new(MockKafkaProducer)
			mockConfigLoader := new(MockConfigLoader)

			tt.setupMocks(mockExecutor, mockProducer)

			orchestrator := NewOrchestrator(
				mockConfigLoader,
				mockConfig,
				mockExecutor,
				mockProducer,
				true,
				tt.timeout,
			)

			err := orchestrator.Run()

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			mockExecutor.AssertExpectations(t)
			mockProducer.AssertExpectations(t)
			mockConfigLoader.AssertExpectations(t)
		})
	}
}
