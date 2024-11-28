package integration

import (
	"testing"

	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"

	"github.com/stretchr/testify/require"
)

func TestIntegrationPipeline(t *testing.T) {
	// Load the configuration directly using the config package
	configPath := "../../tests/integration/configs/test-app-config.yaml"
	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err, "Configuration should load without errors")

	// Validate the loaded configuration (optional, based on test needs)
	require.NotNil(t, cfg, "Configuration should not be nil")
	require.NotEmpty(t, cfg.App.Kafka.Brokers, "Kafka brokers should be defined")
	require.NotEmpty(t, cfg.App.Postgres.URL, "Postgres URL should be defined")

	// Use RealCommandExecutor to actually run the commands
	executor := &execute_pipeline.RealCommandExecutor{}

	// Execute the pipeline
	err = execute_pipeline.ExecutePipeline(configPath, executor)
	require.NoError(t, err, "Pipeline execution should complete without errors")
}
