package utils

import (
	"syscall"
	"testing"
	"time"
)

func TestSetupSignalHandler(t *testing.T) {
	// Create a signal handler
	sigChan := SetupSignalHandler()

	// Simulate sending a SIGTERM signal
	go func() {
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	// Verify the signal is received
	select {
	case sig := <-sigChan:
		if sig != syscall.SIGTERM {
			t.Errorf("Expected SIGTERM, got %v", sig)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Signal not received")
	}
}
