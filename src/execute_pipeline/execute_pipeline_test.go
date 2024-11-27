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

func TestExecutePipeline(t *testing.T) {
	// Setup
	configPath := "testdata/config.yaml"
	mockExecutor := new(MockCommandExecutor)

	// Create test directory and config file
	err := os.MkdirAll("testdata", 0755)
	assert.NoError(t, err)

	testConfig := `
app:
  profiling: false
  source:
    parser: "json"
    file: "messages.json"
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
	assert.NoError(t, err)

	// Create test pipeline template
	err = os.MkdirAll("pipelines/benthos", 0755)
	assert.NoError(t, err)

	templateContent := `
inputs:
  kafka:
    brokers: ${KAFKA_BROKERS}
    topics: ${KAFKA_TOPICS}
outputs:
  postgres:
    url: ${POSTGRES_URL}
    table: ${POSTGRES_TABLE}
`
	err = os.WriteFile("pipelines/benthos/pipeline.yaml", []byte(templateContent), 0644)
	assert.NoError(t, err)

	// Set expectations
	mockExecutor.On("Execute", "rpk", "connect", "run", "pipelines/benthos/generated-pipeline.yaml").Return(nil)

	// Execute test
	err = ExecutePipeline(configPath, mockExecutor)

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertExpectations(t)

	// Verify generated pipeline file
	generatedContent, err := os.ReadFile("pipelines/benthos/generated-pipeline.yaml")
	assert.NoError(t, err)
	assert.Contains(t, string(generatedContent), "localhost:9092")
	assert.Contains(t, string(generatedContent), "test-topic")

	// Cleanup
	os.RemoveAll("testdata")
	os.RemoveAll("pipelines")
}

func TestGeneratePipelineFile(t *testing.T) {
	// Setup
	configPath := "testdata/config.yaml"
	outputPath := "testdata/generated-pipeline.yaml"

	err := os.MkdirAll("testdata", 0755)
	assert.NoError(t, err)

	testConfig := `
app:
  profiling: false
  source:
    parser: "json"
    file: "messages.json"
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
	assert.NoError(t, err)

	// Create pipeline template
	err = os.MkdirAll("pipelines/benthos", 0755)
	assert.NoError(t, err)

	templateContent := `
inputs:
  kafka:
    brokers: ${KAFKA_BROKERS}
    topics: ${KAFKA_TOPICS}
outputs:
  postgres:
    url: ${POSTGRES_URL}
    table: ${POSTGRES_TABLE}
`
	err = os.WriteFile("pipelines/benthos/pipeline.yaml", []byte(templateContent), 0644)
	assert.NoError(t, err)

	// Execute test
	err = GeneratePipelineFile(configPath, outputPath)

	// Assertions
	assert.NoError(t, err)

	generatedContent, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Contains(t, string(generatedContent), "localhost:9092")
	assert.Contains(t, string(generatedContent), "test-topic")
	assert.Contains(t, string(generatedContent), "postgresql://user:pass@localhost:5432/db")
	assert.Contains(t, string(generatedContent), "test_table")

	// Cleanup
	os.RemoveAll("testdata")
	os.RemoveAll("pipelines")
}

func TestGeneratePipelineFile_MissingPlaceholder(t *testing.T) {
	// Setup
	configPath := "testdata/config.yaml"
	outputPath := "testdata/generated-pipeline.yaml"
	templatePath := "pipelines/benthos/pipeline.yaml"

	// Create test directories
	err := os.MkdirAll("testdata", 0755)
	assert.NoError(t, err)
	err = os.MkdirAll("pipelines/benthos", 0755)
	assert.NoError(t, err)

	// Create test configuration
	testConfig := `
app:
  profiling: false
  source:
    parser: "json"
    file: "messages.json"
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
	assert.NoError(t, err)

	// Create a pipeline template with a missing placeholder
	mockTemplate := `
inputs:
  kafka:
    brokers: ${KAFKA_BROKERS}
    topics: ${KAFKA_TOPICS} # Placeholder intentionally missing from config
outputs:
  postgres:
    url: ${POSTGRES_URL}
    table: ${POSTGRES_TABLE}
`
	err = os.WriteFile(templatePath, []byte(mockTemplate), 0644)
	assert.NoError(t, err)

	// Attempt to generate the pipeline file
	err = GeneratePipelineFile(configPath, outputPath)

	// Assertions: Expect an error due to the missing placeholder
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "placeholder ${KAFKA_TOPICS} is not set")

	// Cleanup
	os.RemoveAll("testdata")
	os.RemoveAll("pipelines")
}
