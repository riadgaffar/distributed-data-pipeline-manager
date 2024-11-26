package main

import (
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/parsers"
	"distributed-data-pipeline-manager/src/producer"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
)

// Profiling files
var cpuProfileFile *os.File
var memProfileFile *os.File

func main() {
	// Command-line flags
	jsonFilePath := flag.String("json-file", "messages.json", "Path to the JSON file for dynamic input")
	configPath := flag.String("config", "pipelines/benthos/sample-pipeline.yaml", "Path to the pipeline configuration file")
	profile := flag.Bool("profile", false, "Enable profiling")
	flag.Parse()

	log.Println("INFO: Distributed Data Pipeline Manager")

	// Enable profiling if requested
	if *profile {
		enableCPUProfiling()
		defer stopCPUProfiling()
		defer writeMemoryProfile()
	}

	// Capture termination signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("INFO: Caught termination signal, shutting down...")
		if *profile {
			stopCPUProfiling()
			writeMemoryProfile()
		}
		os.Exit(0)
	}()

	// Step 1: Parse the pipeline configuration
	log.Printf("DEBUG: Parsing pipeline configuration from: %s\n", *configPath)
	pipelineConfig, err := config.ParsePipelineConfig(*configPath)
	if err != nil {
		log.Fatalf("ERROR: Failed to parse pipeline configuration: %v\n", err)
	}
	log.Printf("DEBUG: Parsed input configuration: %+v\n", pipelineConfig.Input)

	// Step 2: Read JSON file for input messages
	log.Printf("DEBUG: Reading JSON file: %s\n", *jsonFilePath)
	data, err := readJSONFile(*jsonFilePath)
	if err != nil {
		log.Fatalf("ERROR: Failed to read JSON file: %v\n", err)
	}

	// Step 3: Initialize the JSON parser
	parser := &parsers.JSONParser{}

	// Step 4: Start the producer
	log.Println("DEBUG: Starting producer...")
	brokers := pipelineConfig.Input.Kafka.Addresses // Use brokers from config
	topics := pipelineConfig.Input.Kafka.Topics     // Use topics from config
	count := 5                                      // Example limit on messages

	if err := producer.ProduceMessages(brokers, topics, count, parser, data); err != nil {
		log.Fatalf("ERROR: Failed to produce messages: %v\n", err)
	}

	// Step 5: Execute the pipeline
	log.Println("DEBUG: Executing pipeline...")
	executor := &execute_pipeline.RealCommandExecutor{} // Use the real command executor
	if err := execute_pipeline.ExecutePipeline(*configPath, executor); err != nil {
		log.Fatalf("ERROR: Pipeline execution failed: %v\n", err)
	}

	log.Println("INFO: Pipeline executed successfully.")
}

// enableCPUProfiling starts CPU profiling.
func enableCPUProfiling() {
	var err error
	cpuProfileFile, err = os.Create("cpu.pprof")
	if err != nil {
		log.Fatalf("ERROR: Failed to create CPU profile: %v\n", err)
	}
	pprof.StartCPUProfile(cpuProfileFile)
	log.Println("DEBUG: CPU profiling enabled")
}

// stopCPUProfiling stops CPU profiling.
func stopCPUProfiling() {
	if cpuProfileFile != nil {
		pprof.StopCPUProfile()
		cpuProfileFile.Close()
		log.Println("DEBUG: CPU profiling stopped")
	}
}

// writeMemoryProfile writes the memory profile to a file.
func writeMemoryProfile() {
	var err error
	memProfileFile, err = os.Create("mem.pprof")
	if err != nil {
		log.Printf("ERROR: Failed to create memory profile: %v\n", err)
		return
	}
	defer memProfileFile.Close()

	if err := pprof.WriteHeapProfile(memProfileFile); err != nil {
		log.Printf("ERROR: Failed to write memory profile: %v", err)
	} else {
		log.Println("DEBUG: Memory profiling written to mem.pprof")
	}
}

// readJSONFile reads and returns the contents of a JSON file.
func readJSONFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return data, nil
}
