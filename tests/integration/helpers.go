package integration

import (
	"database/sql"
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/parsers"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type ProcessedData struct {
	ID        string    // UUID
	Timestamp time.Time // Timestamp
	Data      string    // Actual processed data
}

// PipelineTestHelper encapsulates the lifecycle of a pipeline integration test
type PipelineTestHelper struct {
	commandExecutor execute_pipeline.CommandExecutor // Interface for command execution
	configPath      string
	pipelineCfg     string
}

// NewPipelineTestHelper initializes the helper with required paths
func NewPipelineTestHelper(configPath, pipelineCfg string, commandExecutor execute_pipeline.CommandExecutor) *PipelineTestHelper {
	return &PipelineTestHelper{
		configPath:      configPath,
		pipelineCfg:     pipelineCfg,
		commandExecutor: commandExecutor,
	}
}

// Load TestConfig loads the test configuration.
func loadTestConfig(t *testing.T, configPath string) *config.AppConfig {
	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err, "Failed to load test configuration")
	return cfg
}

// parseTestMessages parses the test messages from the specified file.
// parseTestMessages parses the test messages from the specified file.
func parseTestMessages(t *testing.T, testDataPath string) []interface{} {
	parser := &parsers.JSONParser{}
	data, err := os.ReadFile(testDataPath)
	require.NoError(t, err, "Failed to read test data file")

	parsedData, err := parser.Parse(data)
	require.NoError(t, err, "Failed to parse test messages")
	require.NotNil(t, parsedData, "Parsed data should not be nil")

	// Handle different types of parsed data
	switch v := parsedData.(type) {
	case []interface{}:
		return v
	case map[string]interface{}:
		return []interface{}{v}
	default:
		return []interface{}{parsedData}
	}
}

// produceMessagesToKafka sends test messages to Kafka topics.
func produceMessagesToKafka(t *testing.T, cfg *config.AppConfig, messages []interface{}) {
	// Ensure brokers and topics are defined
	require.NotEmpty(t, cfg.App.Kafka.Brokers, "Kafka brokers are not defined in the config")
	require.NotEmpty(t, cfg.App.Kafka.Topics, "Kafka topics are not defined in the config")

	// Use the first topic from the config for simplicity
	topic := cfg.App.Kafka.Topics[0]

	// Create a Kafka producer
	producer, err := createKafkaProducer(cfg.App.Kafka.Brokers)
	require.NoError(t, err, "Failed to create Kafka producer")
	defer producer.Close()

	// Produce messages
	for i, msg := range messages {
		key := fmt.Sprintf("key-%d", i)
		err := produceMessage(producer, topic, key, msg)
		require.NoError(t, err, "Failed to produce message to Kafka")
	}

	t.Logf("Successfully produced %d messages to Kafka topic '%s'", len(messages), topic)
}

// createKafkaProducer initializes a Kafka producer.
func createKafkaProducer(brokers []string) (*kafka.Producer, error) {
	config := kafka.ConfigMap{
		"bootstrap.servers": strings.Join(brokers, ","),
	}
	return kafka.NewProducer(&config)
}

// produceMessage sends a single message to the Kafka topic.
func produceMessage(producer *kafka.Producer, topic, key string, value interface{}) error {
	msgBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	deliveryChan := make(chan kafka.Event, 1)
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          msgBytes,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery report
	e := <-deliveryChan
	m := e.(*kafka.Message)
	close(deliveryChan)

	if m.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
	}

	return nil
}

// startPipeline starts the pipeline and returns a helper for controlling it.
func startPipeline(t *testing.T, cfg *config.AppConfig, executor execute_pipeline.CommandExecutor) *PipelineTestHelper {
	require.NotEmpty(t, cfg.App.GeneratePipelineConfig, "Pipeline configuration path is not set")
	require.NotEmpty(t, cfg.App.PipelineTemplate, "Pipeline template path is not set")

	// Initialize the pipeline helper with the required arguments
	helper := NewPipelineTestHelper(cfg.App.GeneratePipelineConfig, cfg.App.PipelineTemplate, executor)

	// Start the pipeline asynchronously
	go func() {
		t.Logf("Starting pipeline using config: %s", cfg.App.GeneratePipelineConfig)
		err := execute_pipeline.ExecutePipeline(os.Getenv("CONFIG_PATH"), executor)
		if err != nil {
			t.Errorf("Pipeline execution failed: %v", err)
		}
	}()

	// Wait for the pipeline to initialize
	time.Sleep(10 * time.Second)

	t.Log("Pipeline started successfully")
	return helper
}

// validateProcessedData verifies the number of processed messages in the database.
func validateProcessedData(t *testing.T, cfg *config.AppConfig, expectedCount int) {
	processedData := fetchAndVerifyProcessedData(t, cfg)
	require.Len(t, processedData, expectedCount, "Unexpected number of processed messages in the database")
}

func fetchAndVerifyProcessedData(t *testing.T, cfg *config.AppConfig) []ProcessedData {
	processedData, err := fetchProcessedData(cfg.App.Postgres.URL, cfg.App.Postgres.Table)
	require.NoError(t, err, "Failed to fetch processed data from database")
	return processedData
}

// Fetch processed data from Postgres
func fetchProcessedData(url, table string) ([]ProcessedData, error) {
	// Register the Postgres driver with database/sql
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT id, timestamp, data FROM %s", table)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProcessedData
	for rows.Next() {
		var data ProcessedData
		err := rows.Scan(&data.ID, &data.Timestamp, &data.Data)
		if err != nil {
			return nil, err
		}
		results = append(results, data)
	}

	return results, nil
}

// StopPipeline gracefully stops the pipeline process
func (p *PipelineTestHelper) StopPipeline(t *testing.T) {
	if p.commandExecutor == nil {
		t.Log("No executor provided to stop the pipeline!")
		return
	}

	t.Log("Waiting for graceful shutdown of the pipeline process...")

	// Use the executor to stop the pipeline process
	err := p.commandExecutor.StopPipeline()
	require.NoError(t, err, "Failed to stop the pipeline process")
}
