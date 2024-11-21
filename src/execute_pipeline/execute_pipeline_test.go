package execute_pipeline

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockCommandExecutor is a mock implementation of CommandExecutor.
type MockCommandExecutor struct {
	ExpectedError error
}

func (m *MockCommandExecutor) Execute(name string, args ...string) error {
	// Simulate command execution behavior.
	return m.ExpectedError
}

func TestExecutePipeline_Success(t *testing.T) {
	// Use the mock executor with no error
	mockExecutor := &MockCommandExecutor{ExpectedError: nil}

	// Call ExecutePipeline
	err := ExecutePipeline("/path/to/config.yaml", mockExecutor)

	// Assert no error
	assert.NoError(t, err)
}

func TestExecutePipeline_Failure(t *testing.T) {
	// Use the mock executor with an error
	mockExecutor := &MockCommandExecutor{ExpectedError: errors.New("mocked failure")}

	// Call ExecutePipeline
	err := ExecutePipeline("/path/to/config.yaml", mockExecutor)

	// Assert error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mocked failure")
}
