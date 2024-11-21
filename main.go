package main

import (
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/producer"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
)

func main() {
	// CPU Profiling
	f, err := os.Create("cpu.pprof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	log.Println("Distributed Data Pipeline Manager")

	// Debug: Parsing pipeline configuration
	fmt.Println("DEBUG: Parsing pipeline configuration...")
	configPath := "pipelines/benthos/sample-pipeline.yaml"
	fmt.Printf("DEBUG: Parsing pipeline configuration from: %s\n", configPath)
	pipelineConfig, err := config.ParsePipelineConfig(configPath)
	if err != nil {
		log.Fatalf("ERROR: Failed to parse pipeline: %v\n", err)
	}
	fmt.Printf("DEBUG: Parsed input configuration: %+v\n", pipelineConfig.Input)

	// Debug: Starting producer
	fmt.Println("DEBUG: Starting producer...")
	// Extract required values from pipelineConfig
	brokers := pipelineConfig.Input.Kafka.Addresses
	topic := pipelineConfig.Input.Kafka.Topics

	// Call ProduceMessages with the dynamically loaded configuration
	err = producer.ProduceMessages(brokers, topic, 10) // Produce 10 messages
	if err != nil {
		log.Fatalf("ERROR: Failed to produce messages: %v\n", err)
		os.Exit(1)
	}

	// Execute the pipeline
	fmt.Println("DEBUG: Executing pipeline...")
	executor := &execute_pipeline.RealCommandExecutor{} // Use the real command executor
	err = execute_pipeline.ExecutePipeline(configPath, executor)
	if err != nil {
		log.Fatalf("ERROR: Pipeline execution failed: %v\n", err)
	}

	fmt.Println("Pipeline executed successfully.")

	// Memory Profiling
	m, err := os.Create("mem.pprof")
	if err != nil {
		panic(err)
	}
	pprof.WriteHeapProfile(m)
	m.Close()
}
