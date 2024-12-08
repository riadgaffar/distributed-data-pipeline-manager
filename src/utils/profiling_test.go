package utils

import (
	"os"
	"testing"
)

func TestEnableProfiling(t *testing.T) {
	cpuProfile := "test_cpu.pprof"
	memProfile := "test_mem.pprof"

	// Start profiling
	cleanup := EnableProfiling(cpuProfile, memProfile)

	// Check if profiling files are created
	defer os.Remove(cpuProfile)
	defer os.Remove(memProfile)

	// Stop profiling
	cleanup()

	// Verify file existence
	if _, err := os.Stat(cpuProfile); os.IsNotExist(err) {
		t.Errorf("CPU profile not created: %v", err)
	}

	if _, err := os.Stat(memProfile); os.IsNotExist(err) {
		t.Errorf("Memory profile not created: %v", err)
	}
}
