package bootstrap

import (
	"context"
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/parsers"
	"distributed-data-pipeline-manager/src/producer"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaAdminClient struct {
	client *kafka.AdminClient
}

// NewKafkaAdminClient creates a new Kafka admin client that conforms to producer.AdminClient.
func NewKafkaAdminClient(brokers string) (producer.AdminClient, error) {
	client, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": brokers})
	if err != nil {
		return nil, err
	}
	return &KafkaAdminClient{client: client}, nil
}

// GetMetadata fetches metadata for a topic.
func (k *KafkaAdminClient) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
	return k.client.GetMetadata(topic, allTopics, timeoutMs)
}

// CreatePartitions increases the number of partitions for a topic.
func (k *KafkaAdminClient) CreatePartitions(ctx context.Context, partitionsSpec []kafka.PartitionsSpecification, opts ...kafka.CreatePartitionsAdminOption) ([]kafka.TopicResult, error) {
	return k.client.CreatePartitions(ctx, partitionsSpec, opts...)
}

// Close closes the admin client.
func (k *KafkaAdminClient) Close() {
	k.client.Close()
}

// InitializeApp initializes the application configuration and validates inputs.
func InitializeApp(configPath string, port int) (*config.AppConfig, error) {
	// Validate port first
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("port number %d is out of range (1-65535)", port)
	}

	// Load configuration (this handles path precedence)
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	log.Printf("INFO: Application initialized with config: %s on port %d\n", configPath, port)
	return cfg, nil
}

// InitializeParser creates a parser based on the configuration.
func InitializeParser(parserType string) (parsers.Parser, error) {
	switch parserType {
	case "json":
		return &parsers.JSONParser{}, nil
	case "avro":
		return nil, fmt.Errorf("parser type '%s' is not yet implemented", parserType)
	case "parquet":
		return nil, fmt.Errorf("parser type '%s' is not yet implemented", parserType)
	default:
		return nil, fmt.Errorf("unsupported parser type: '%s'", parserType)
	}
}

// InitializeKafka ensures Kafka topics are properly configured and starts monitoring.
func InitializeKafka(cfg *config.AppConfig, adminClientFactory func(string) (producer.AdminClient, error)) (producer.AdminClient, error) {
	brokers := strings.Join(cfg.App.Kafka.Brokers, ",")
	adminClient, err := adminClientFactory(brokers)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka AdminClient: %w", err)
	}

	// Ensure topic partitions
	for _, topic := range cfg.App.Kafka.Topics {
		if err := producer.EnsureTopicPartitions(adminClient, topic, 3); err != nil {
			adminClient.Close() // Ensure cleanup on error
			return nil, fmt.Errorf("failed to ensure topic partitions for topic %s: %w", topic, err)
		}
	}

	log.Println("INFO: Kafka AdminClient initialized and topic partitions ensured.")
	return adminClient, nil
}

func loadConfig(configPathFlag string) (*config.AppConfig, error) {
	// Determine configuration path precedence
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = configPathFlag
		if configPath == "" {
			return nil, fmt.Errorf("configuration path is required: neither CONFIG_PATH env var nor --config flag provided")
		}
		log.Printf("INFO: CONFIG_PATH not set, using --config flag value: %s", configPath)
	} else {
		log.Printf("INFO: Using CONFIG_PATH environment variable: %s", configPath)
	}

	// Validate the configuration path
	if err := validateConfigPath(configPath); err != nil {
		return nil, fmt.Errorf("invalid configuration path: %w", err)
	}

	// Load the configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from %s: %w", configPath, err)
	}

	return cfg, nil
}

func validateConfigPath(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", configPath)
	}
	return nil
}
