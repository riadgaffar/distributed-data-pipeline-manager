package integration

import (
	"os"
	"testing"
	"time"

	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/parsers"

	"github.com/stretchr/testify/require"
)

func TestIntegrationPipeline(t *testing.T) {
	// Load configuration for integration test
	configPath := "../../tests/integration/configs/test-app-config.yaml"
	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err, "Failed to load test configuration")

	// Initialize JSON parser
	parser := &parsers.JSONParser{}

	// Read and parse test messages
	testDataPath := cfg.App.Source.File
	data, err := os.ReadFile(testDataPath)
	require.NoError(t, err, "Failed to read test data file")

	messages, err := parser.Parse(data)
	require.NoError(t, err, "Failed to parse test messages")
	require.NotEmpty(t, messages, "Parsed messages should not be empty")

	// Produce messages to Kafka
	topics := cfg.App.Kafka.Topics
	messageCount := len(messages)
	err = produceTestMessages(cfg.App.Kafka.Brokers, topics, messageCount, parser, data)
	require.NoError(t, err, "Failed to produce messages to Kafka")

	// Execute the pipeline in a separate goroutine
	executor := &execute_pipeline.RealCommandExecutor{}
	go func() {
		err := execute_pipeline.ExecutePipeline(configPath, executor)
		require.NoError(t, err, "Pipeline execution should complete without errors")
	}()

	// Wait for pipeline to process the messages
	time.Sleep(10 * time.Second)

	// Fetch and verify processed data
	processedData, err := fetchProcessedData(cfg.App.Postgres.URL, cfg.App.Postgres.Table)
	require.NoError(t, err, "Failed to fetch processed data from database")
	require.Len(t, processedData, 3, "Expected 3 processed messages in the database")
}
