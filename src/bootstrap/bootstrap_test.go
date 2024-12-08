package bootstrap

import (
	"context"
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/producer"
	"errors"
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

		// Write mock config to a file
		err := os.WriteFile("test_config.yaml", []byte(mockConfig), 0644)
		assert.NoError(t, err)
		defer os.Remove("test_config.yaml")

		// Test valid configuration and port
		cfg, err := InitializeApp("test_config.yaml", 8080)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("missing configuration path", func(t *testing.T) {
		// Test with missing configuration path
		_, err := InitializeApp("", 8080)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration path is required")
	})

	t.Run("invalid port", func(t *testing.T) {
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

		// Write mock config to a file
		err := os.WriteFile("test_config.yaml", []byte(mockConfig), 0644)
		assert.NoError(t, err)
		defer os.Remove("test_config.yaml")

		// Test with invalid port
		_, err = InitializeApp("test_config.yaml", 99999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "port number 99999 is out of range")
	})
}

// TestInitializeKafka validates Kafka initialization
func TestInitializeKafka(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		mockClient := new(MockAdminClient)
		mockClient.On("GetMetadata", mock.AnythingOfType("*string"), false, 5000).Return(
			&kafka.Metadata{
				Topics: map[string]kafka.TopicMetadata{
					"test-topic": {Partitions: []kafka.PartitionMetadata{{}, {}}},
				},
			}, nil,
		)
		mockClient.On("CreatePartitions", mock.Anything, mock.Anything, mock.Anything).Return(
			[]kafka.TopicResult{}, nil,
		)
		mockClient.On("Close").Return()

		adminClientFactory := func(brokers string) (producer.AdminClient, error) {
			return mockClient, nil
		}

		mockConfig := &config.AppConfig{
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
					Parser: "parser",
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

		err := InitializeKafka(mockConfig, adminClientFactory)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("failed Kafka admin client creation", func(t *testing.T) {
		adminClientFactory := func(brokers string) (producer.AdminClient, error) {
			return nil, errors.New("mock admin client error")
		}

		mockConfig := &config.AppConfig{
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
					Parser: "parser",
				},
				Kafka: struct {
					Brokers       []string `yaml:"brokers"`
					Topics        []string `yaml:"topics"`
					ConsumerGroup string   `yaml:"consumer_group"`
				}{
					Brokers:       []string{"invalid:9092"},
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

		err := InitializeKafka(mockConfig, adminClientFactory)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock admin client error")
	})

	t.Run("error fetching topic metadata", func(t *testing.T) {
		mockClient := new(MockAdminClient)
		mockClient.On("GetMetadata", mock.AnythingOfType("*string"), false, 5000).Return(
			(*kafka.Metadata)(nil), errors.New("metadata fetch failed"),
		)
		mockClient.On("Close").Return()

		adminClientFactory := func(brokers string) (producer.AdminClient, error) {
			return mockClient, nil
		}

		mockConfig := &config.AppConfig{
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
					Parser: "parser",
				},
				Kafka: struct {
					Brokers       []string `yaml:"brokers"`
					Topics        []string `yaml:"topics"`
					ConsumerGroup string   `yaml:"consumer_group"`
				}{
					Brokers: []string{"localhost:9092"},
					Topics:  []string{"test-topic"},
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

		err := InitializeKafka(mockConfig, adminClientFactory)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get topic metadata")
		mockClient.AssertExpectations(t)
	})
}
