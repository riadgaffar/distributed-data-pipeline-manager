package json_test

import (
	"distributed-data-pipeline-manager/tests/integration/framework"
	"testing"
)

type JSONParserTest struct {
	*framework.BasePluginTest
	messages []interface{}
}

func TestJSONParser(t *testing.T) {
	fw := framework.NewTestFramework(t, "../../configs/test-app-config.yaml")

	test := &JSONParserTest{
		BasePluginTest: framework.NewBasePluginTest(fw),
	}

	t.Run("Setup", func(t *testing.T) {
		if err := test.Setup(); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	})

	t.Run("Execute", func(t *testing.T) {
		if err := test.Execute(); err != nil {
			t.Fatalf("Execution failed: %v", err)
		}
	})

	t.Run("Validate", func(t *testing.T) {
		if err := test.Validate(); err != nil {
			t.Fatalf("Validation failed: %v", err)
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		if err := test.Teardown(); err != nil {
			t.Fatalf("Teardown failed: %v", err)
		}
	})
}

// Setup implements the setup phase
func (t *JSONParserTest) Setup() error {
	// Load test messages
	messages := t.Framework.Helper.ParseJSONTestMessages("../../test_data/test-messages.json")
	t.messages = messages
	t.Framework.Helper.Setup()
	return nil
}

// Execute implements the execution phase
func (t *JSONParserTest) Execute() error {
	// Produce messages to Kafka
	return t.Framework.Helper.ProduceMessagesToKafka(t.messages)
}

// Validate implements the validation phase
func (t *JSONParserTest) Validate() error {
	// Validate processed data in PostgreSQL
	return t.Framework.Helper.ValidateProcessedData(len(t.messages))
}

// Teardown implements the cleanup phase
func (t *JSONParserTest) Teardown() error {
	return t.Framework.Helper.StopPipeline()
}
