package main

import (
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/parsers"
	"distributed-data-pipeline-manager/src/producer"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/sirupsen/logrus"
)

var cpuProfileFile *os.File

func main() {
	log.Println("INFO: Distributed Data Pipeline Manager")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	// Set logging level
	setLogLevel(cfg.App.LoggerConfig.Level)

	// Enable profiling if configured
	if cfg.App.Profiling {
		defer enableProfiling()()
	}

	// Capture termination signals for graceful shutdown
	setupSignalHandler()

	// Initialize the parser
	parser, err := initializeParser(cfg.App.Source.Parser)
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	// Read source data
	data, err := readSourceData(cfg.App.Source.File)
	if err != nil {
		log.Fatalf("ERROR: Failed to read source file: %v\n", err)
	}

	// Parse JSON data with support for mixed formats
	messages, err := ParseMessages(data)
	if err != nil {
		log.Fatalf("ERROR: Failed to parse JSON data: %v\n", err)
	}

	// Start producer
	messageCount := len(messages)
	log.Println("DEBUG: Starting producer...")
	if err := producer.ProduceMessages(cfg.App.Kafka.Brokers, cfg.App.Kafka.Topics, messageCount, parser, data); err != nil {
		log.Fatalf("ERROR: Failed to produce messages: %v\n", err)
	}

	// Execute the pipeline
	log.Println("DEBUG: Executing pipeline...")
	executor := &execute_pipeline.RealCommandExecutor{}
	if err := execute_pipeline.ExecutePipeline(os.Getenv("CONFIG_PATH"), executor); err != nil {
		log.Fatalf("ERROR: Pipeline execution failed: %v\n", err)
	}

	log.Println("INFO: Pipeline executed successfully.")
}

// loadConfig loads the configuration from CONFIG_PATH
func loadConfig() (*config.AppConfig, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		return nil, fmt.Errorf("CONFIG_PATH environment variable is not set")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	log.Printf("INFO: Loaded configuration: %+v\n", cfg)
	return cfg, nil
}

// setLogLevel sets the log level based on configuration
func setLogLevel(level string) {
	logrus.Printf("DEBUG: Received log level: '%s'", level)

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Printf("WARN: Invalid log level '%s', defaulting to INFO\n", level)
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
}

// enableProfiling enables CPU and memory profiling
func enableProfiling() func() {
	var err error
	cpuProfileFile, err = os.Create("cpu.pprof")
	if err != nil {
		log.Fatalf("ERROR: Failed to create CPU profile: %v\n", err)
	}
	pprof.StartCPUProfile(cpuProfileFile)
	log.Println("DEBUG: CPU profiling enabled")

	return func() {
		pprof.StopCPUProfile()
		if cpuProfileFile != nil {
			cpuProfileFile.Close()
		}
		log.Println("DEBUG: CPU profiling stopped")
		writeMemoryProfile()
	}
}

// writeMemoryProfile writes the memory profile to a file
func writeMemoryProfile() {
	memProfileFile, err := os.Create("mem.pprof")
	if err != nil {
		log.Printf("ERROR: Failed to create memory profile: %v\n", err)
		return
	}
	defer memProfileFile.Close()

	if err := pprof.WriteHeapProfile(memProfileFile); err != nil {
		log.Printf("ERROR: Failed to write memory profile: %v\n", err)
	} else {
		log.Println("DEBUG: Memory profiling written to mem.pprof")
	}
}

// setupSignalHandler captures termination signals for graceful shutdown
func setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("INFO: Caught termination signal, shutting down...")
		os.Exit(0)
	}()
}

// initializeParser creates a parser based on the provided type
func initializeParser(parserType string) (parsers.Parser, error) {
	switch parserType {
	case "json":
		return &parsers.JSONParser{}, nil
	case "avro":
		return nil, fmt.Errorf("parser type '%s' is not yet implemented", parserType)
	case "parquet":
		return nil, fmt.Errorf("parser type '%s' is not yet implemented", parserType)
	default:
		return nil, fmt.Errorf("unsupported parser type: '%s'", parserType)
	}
}

// readSourceData reads and returns the contents of a source file
func readSourceData(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file %s: %w", filePath, err)
	}
	return data, nil
}

// ParseMessages parses JSON data with support for mixed formats
// ParseMessages parses JSON data with support for mixed formats
func ParseMessages(data []byte) ([]string, error) {
	var result []string

	// Try parsing as an object with a "messages" field
	var objectWithMessages struct {
		Messages []string `json:"messages"`
	}
	if err := json.Unmarshal(data, &objectWithMessages); err == nil && len(objectWithMessages.Messages) > 0 {
		result = objectWithMessages.Messages
		return result, nil
	}

	// Try parsing as an array of strings
	var arrayOfStrings []string
	if err := json.Unmarshal(data, &arrayOfStrings); err == nil {
		result = arrayOfStrings
		return result, nil
	}

	// Try parsing as a single object (fallback for single item cases)
	var singleMessage map[string]interface{}
	if err := json.Unmarshal(data, &singleMessage); err == nil {
		// Optionally process the single object and decide on the representation
		singleMessageJSON, _ := json.Marshal(singleMessage)
		result = []string{string(singleMessageJSON)}
		return result, nil
	}

	return nil, fmt.Errorf("unsupported JSON format: %s", string(data))
}
