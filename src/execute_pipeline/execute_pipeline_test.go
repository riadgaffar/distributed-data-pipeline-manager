package execute_pipeline

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockCommandExecutor struct {
	Commands []string
	Stopped  bool
}

// Mock implementation of the ExecuteCommand method
func (m *MockCommandExecutor) ExecuteCommand(name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	m.Commands = append(m.Commands, command)
	log.Printf("MockCommandExecutor: Command executed: %s", command)
	return nil
}

func (m *MockCommandExecutor) StopPipeline() error {
	m.Stopped = true
	log.Println("Mock: Pipeline stopped.")
	return nil
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

func validateGeneratedPipeline(t *testing.T, generatedPipelinePath string) {
	// Check if the file exists
	generatedContent, err := os.ReadFile(generatedPipelinePath)
	assert.NoError(t, err, "Generated pipeline file should exist")

	// Verify placeholder replacements
	assert.Contains(t, string(generatedContent), "localhost:9092", "Pipeline file should contain Kafka broker")
	assert.Contains(t, string(generatedContent), "test-topic", "Pipeline file should contain Kafka topic")
	assert.Contains(t, string(generatedContent), "postgresql://user:pass@localhost:5432/db", "Pipeline file should contain Postgres URL")
	assert.Contains(t, string(generatedContent), "test_table", "Pipeline file should contain Postgres table name")
}

func cleanupTestEnv(t *testing.T) {
	err := os.RemoveAll("testdata")
	assert.NoError(t, err, "Failed to remove testdata directory")

	err = os.RemoveAll("pipelines")
	assert.NoError(t, err, "Failed to remove pipelines directory")
}

func TestExecutePipeline(t *testing.T) {
	// Setup paths for test files
	configPath := "testdata/config.yaml"
	templatePath := "pipelines/benthos/pipeline.yaml"
	generatedPipelinePath := "pipelines/benthos/generated-pipeline.yaml"

	// Create a mock executor to simulate command execution
	mockExecutor := new(MockCommandExecutor)

	// Setup test environment (directories and files)
	setupTestEnv(t, configPath, templatePath)
	defer cleanupTestEnv(t) // Cleanup after test

	// Execute the pipeline using the mock executor
	err := ExecutePipeline(configPath, mockExecutor)

	// Assertions
	assert.NoError(t, err, "ExecutePipeline should not return an error")

	// Verify expectations on the mocked executor
	assert.Equal(t, 1, len(mockExecutor.Commands), "Mock executor should execute exactly one command")
	assert.Contains(t, mockExecutor.Commands[0], "rpk connect run", "Command should contain the expected 'rpk connect run'")
	assert.Contains(t, mockExecutor.Commands[0], generatedPipelinePath, "Command should use the generated pipeline path")

	// Validate the generated pipeline configuration file
	validateGeneratedPipeline(t, generatedPipelinePath)
}

func TestGeneratePipelineFile(t *testing.T) {
	// Setup paths
	configPath := "testdata/config.yaml"
	outputPath := "testdata/generated-pipeline.yaml"
	templatePath := "pipelines/benthos/pipeline.yaml"

	// Setup test environment
	setupTestEnv(t, configPath, templatePath)
	defer cleanupTestEnv(t)

	// Execute pipeline generation
	err := GeneratePipelineFile(configPath, outputPath)

	// Assertions
	assert.NoError(t, err, "Pipeline generation should succeed")
	assert.FileExists(t, outputPath, "Generated pipeline file should exist")

	// Validate file content
	generatedContent, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Contains(t, string(generatedContent), "localhost:9092")
	assert.Contains(t, string(generatedContent), "test-topic")
	assert.Contains(t, string(generatedContent), "postgresql://user:pass@localhost:5432/db")
	assert.Contains(t, string(generatedContent), "test_table")
}

func TestGeneratePipelineFile_MissingPlaceholder(t *testing.T) {
	// Setup paths
	configPath := "testdata/config.yaml"
	outputPath := "testdata/generated-pipeline.yaml"
	templatePath := "pipelines/benthos/pipeline.yaml"

	// Create necessary directories
	err := os.MkdirAll("testdata", 0755)
	assert.NoError(t, err, "Failed to create testdata directory")
	err = os.MkdirAll("pipelines/benthos", 0755)
	assert.NoError(t, err, "Failed to create pipelines/benthos directory")

	// Create test configuration with missing placeholder
	testConfig := `
app:
  profiling: false
  pipeline_template: pipelines/benthos/pipeline.yaml
  generated_pipeline_config: pipelines/benthos/generated-pipeline.yaml
  kafka:
    brokers: ["localhost:9092"]
    consumer_group: "test-group"
  postgres:
    url: "postgresql://user:pass@localhost:5432/db"
    table: "test_table"
  logger:
    level: "DEBUG"
`
	err = os.WriteFile(configPath, []byte(testConfig), 0644)
	assert.NoError(t, err, "Failed to write test configuration file")

	// Create pipeline template with a missing placeholder
	mockTemplate := `
input:
  kafka:
    addresses: ["${KAFKA_BROKERS}"]
    topics: ["${KAFKA_TOPICS}"] # Placeholder intentionally missing in config
output:
  database:
    driver: postgres
    dsn: "${POSTGRES_URL}"
    table: "${POSTGRES_TABLE}"
`
	err = os.WriteFile(templatePath, []byte(mockTemplate), 0644)
	assert.NoError(t, err, "Failed to write pipeline template file")

	// Attempt to generate the pipeline file
	err = GeneratePipelineFile(configPath, outputPath)

	// Assertions: Expect an error due to the missing placeholder
	assert.Error(t, err, "Expected error due to missing placeholder in pipeline template")
	assert.Contains(t, err.Error(), "${KAFKA_TOPICS}", "Error should mention the missing placeholder")

	// Ensure the output file is not created
	_, fileErr := os.Stat(outputPath)
	assert.True(t, os.IsNotExist(fileErr), "Generated pipeline file should not exist due to error")

	// Cleanup
	cleanupTestEnv(t)
}
