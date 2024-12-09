package utils

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupSignalHandler(t *testing.T) {
	var (
		exitCalled bool
		exitCode   int
		wg         sync.WaitGroup
	)

	// Override exitFunc for testing
	exitFunc = func(code int) {
		exitCalled = true
		exitCode = code
		wg.Done() // Notify test that exitFunc was called
	}

	// Add a wait group counter
	wg.Add(1)

	// Call SetupSignalHandler
	SetupSignalHandler()

	// Simulate sending a termination signal
	go func() {
		signalChan <- os.Interrupt
	}()

	// Wait for the signal handler to execute
	wg.Wait()

	// Assert that exitFunc was called
	assert.True(t, exitCalled, "exitFunc should have been called")
	assert.Equal(t, 0, exitCode, "exitFunc should have been called with code 0")
}
