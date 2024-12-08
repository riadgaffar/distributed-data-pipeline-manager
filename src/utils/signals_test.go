package utils

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSetupSignalHandler(t *testing.T) {
	// Override ExitFunc to prevent os.Exit from terminating the test
	called := false
	ExitFunc = func(code int) {
		called = true
		if code != 0 {
			t.Errorf("Expected exit code 0, got %d", code)
		}
	}

	// Reset ExitFunc after the test
	defer func() { ExitFunc = os.Exit }()

	// Set up the signal handler
	SetupSignalHandler()

	// Simulate sending a termination signal
	go func() {
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	// Wait for the signal to be handled
	time.Sleep(200 * time.Millisecond)

	// Assert that ExitFunc was called
	if !called {
		t.Error("ExitFunc was not called as expected")
	}
}
