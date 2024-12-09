package integration

import (
	"testing"

	"distributed-data-pipeline-manager/src/execute_pipeline"
)

// TestIntegrationPipeline validates the complete pipeline execution.
func TestIntegrationPipelineJSON(t *testing.T) {
	// Step 1: Load configuration
	cfg := loadTestConfig(t, "../../tests/integration/configs/test-app-config.yaml")

	// Step 2: Parse JSON test messages
	messages := parseJSONTestMessages(t, "../../tests/integration/test_data/test-messages.json")

	// Step 3: Produce JSON messages to Kafka
	produceMessagesToKafka(t, cfg, messages)

	// Step 4: Execute the pipeline
	executor := &execute_pipeline.RealCommandExecutor{}
	helper := startPipeline(t, cfg, executor)
	defer helper.StopPipeline(t)

	// Step 5: Validate processed messages in the database
	validateProcessedData(t, cfg, len(messages))
}
