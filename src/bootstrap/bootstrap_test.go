package bootstrap

import (
	"context"
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/producer"
	"fmt"
	"os"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAdminClient simulates Kafka admin client behavior for tests
type MockAdminClient struct {
	mock.Mock
}

func (m *MockAdminClient) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
	args := m.Called(topic, allTopics, timeoutMs)
	return args.Get(0).(*kafka.Metadata), args.Error(1)
}

func (m *MockAdminClient) CreatePartitions(ctx context.Context, partitionsSpec []kafka.PartitionsSpecification, opts ...kafka.CreatePartitionsAdminOption) ([]kafka.TopicResult, error) {
	args := m.Called(ctx, partitionsSpec, opts)
	return args.Get(0).([]kafka.TopicResult), args.Error(1)
}

func (m *MockAdminClient) Close() {
	m.Called()
}

// TestInitializeApp validates app initialization
func TestInitializeApp(t *testing.T) {
	t.Run("valid configuration and port", func(t *testing.T) {
		mockConfig := `
app:
  profiling: false
  pipeline_template: "template"
  generated_pipeline_config: "config"
  source:
    parser: "json"
  kafka:
    brokers:
      - "localhost:9092"
    topics:
      - "test-topic"
    consumer_group: "test-group"
  postgres:
    url: "postgres://localhost:5432"
    table: "table"
  logger:
    level: "info"
`
		// Create temporary config file
		tmpfile, err := os.CreateTemp("", "test_config*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.Write([]byte(mockConfig))
		assert.NoError(t, err)
		tmpfile.Close()

		// Test valid configuration and port
		cfg, err := InitializeApp(tmpfile.Name(), 8080)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("missing configuration path", func(t *testing.T) {
		cfg, err := InitializeApp("", 8080)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "configuration path is required")
	})

	t.Run("invalid port", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_config*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		cfg, err := InitializeApp(tmpfile.Name(), 99999)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "port number 99999 is out of range")
	})

	t.Run("non-existent config file", func(t *testing.T) {
		cfg, err := InitializeApp("/non/existent/path.yaml", 8080)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "invalid configuration path")
	})
}

// TestInitializeKafka validates Kafka initialization
func TestInitializeKafka(t *testing.T) {
	// Shared config for all test cases
	mockConfig := &config.AppConfig{
		App: struct {
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
		}{
			Profiling:              false,
			PipelineTemplate:       "template",
			GeneratePipelineConfig: "config",
			Source: struct {
				Parser     string `yaml:"parser"`
				PluginPath string `yaml:"plugin_path"`
			}{
				Parser:     "json",
				PluginPath: "bin/plugins/json.so",
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

	t.Run("successful initialization", func(t *testing.T) {
		mockClient := new(MockAdminClient)

		mockClient.On("GetMetadata", mock.AnythingOfType("*string"), false, 5000).Return(
			&kafka.Metadata{
				Topics: map[string]kafka.TopicMetadata{
					"test-topic": {Partitions: []kafka.PartitionMetadata{{}, {}}},
				},
			}, nil,
		).Once()

		mockClient.On("CreatePartitions", mock.Anything, []kafka.PartitionsSpecification{
			{Topic: "test-topic", IncreaseTo: 3},
		}, mock.Anything).Return(
			[]kafka.TopicResult{}, nil,
		).Once()

		adminClient, err := InitializeKafka(mockConfig, func(brokers string) (producer.AdminClient, error) {
			return mockClient, nil
		})

		assert.NoError(t, err)
		assert.NotNil(t, adminClient)
		mockClient.AssertExpectations(t)
	})

	t.Run("error case calls Close", func(t *testing.T) {
		mockClient := new(MockAdminClient)

		mockClient.On("GetMetadata", mock.AnythingOfType("*string"), false, 5000).Return(
			&kafka.Metadata{}, fmt.Errorf("metadata error"),
		).Once()

		mockClient.On("Close").Return().Once()

		adminClient, err := InitializeKafka(mockConfig, func(brokers string) (producer.AdminClient, error) {
			return mockClient, nil
		})

		assert.Error(t, err)
		assert.Nil(t, adminClient)
		assert.Contains(t, err.Error(), "failed to ensure topic partitions")
		mockClient.AssertExpectations(t)
	})
}
