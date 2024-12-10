package framework

import (
	"testing"
	"time"
)

type TestFramework struct {
	Config     *TestConfig
	Helper     *PipelineTestHelper
	Assertions *TestAssertions
}

type TestConfig struct {
	WaitTime        time.Duration
	RetryAttempts   int
	TimeoutDuration time.Duration
}

func NewTestFramework(t *testing.T, configPath string) *TestFramework {
	return &TestFramework{
		Config: &TestConfig{
			WaitTime:        10 * time.Second,
			RetryAttempts:   3,
			TimeoutDuration: 60 * time.Second,
		},
		Helper:     NewPipelineTestHelper(configPath),
		Assertions: NewTestAssertions(t),
	}
}
