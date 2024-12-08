package utils

import (
	"log"
	"os"
	"runtime/pprof"
)

// EnableProfiling enables CPU and memory profiling and returns a cleanup function.
func EnableProfiling(cpuProfilePath, memProfilePath string) func() {
	cpuFile, err := os.Create(cpuProfilePath)
	if err != nil {
		log.Fatalf("ERROR: Failed to create CPU profile: %v", err)
	}
	pprof.StartCPUProfile(cpuFile)
	log.Println("DEBUG: CPU profiling enabled")

	return func() {
		pprof.StopCPUProfile()
		cpuFile.Close()
		log.Println("DEBUG: CPU profiling stopped")

		memFile, err := os.Create(memProfilePath)
		if err != nil {
			log.Printf("ERROR: Failed to create memory profile: %v", err)
			return
		}
		defer memFile.Close()

		if err := pprof.WriteHeapProfile(memFile); err != nil {
			log.Printf("ERROR: Failed to write memory profile: %v", err)
		} else {
			log.Println("DEBUG: Memory profiling written")
		}
	}
}
