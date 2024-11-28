package execute_pipeline

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Execute(name string, args ...string) error {
	args_interface := make([]interface{}, len(args)+1)
	args_interface[0] = name
	for i, arg := range args {
		args_interface[i+1] = arg
	}
	return m.Called(args_interface...).Error(0)
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
  pipeline_template_path: pipelines/benthos/
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
	// Setup paths
	configPath := "testdata/config.yaml"
	templatePath := "pipelines/benthos/pipeline.yaml"
	generatedPipelinePath := "pipelines/benthos/generated-pipeline.yaml"

	// Mock executor
	mockExecutor := new(MockCommandExecutor)

	// Setup test environment
	setupTestEnv(t, configPath, templatePath)

	// Mock rpk execution
	mockExecutor.On("Execute", "rpk", "connect", "run", generatedPipelinePath).Return(nil)

	// Execute the pipeline
	err := ExecutePipeline(configPath, mockExecutor)

	// Assertions
	assert.NoError(t, err, "ExecutePipeline should not return an error")
	mockExecutor.AssertExpectations(t)

	// Verify generated pipeline file
	validateGeneratedPipeline(t, generatedPipelinePath)

	// Cleanup
	cleanupTestEnv(t)
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
  pipeline_template_path: pipelines/benthos/
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
