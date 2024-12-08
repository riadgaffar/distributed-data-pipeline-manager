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
	defer cleanup()

	// Check if profiling files are created
	defer os.Remove(cpuProfile)
	defer os.Remove(memProfile)

	// Stop profiling and check file existence
	cleanup()

	if _, err := os.Stat(cpuProfile); os.IsNotExist(err) {
		t.Errorf("CPU profile not created: %v", err)
	}

	if _, err := os.Stat(memProfile); os.IsNotExist(err) {
		t.Errorf("Memory profile not created: %v", err)
	}
}
