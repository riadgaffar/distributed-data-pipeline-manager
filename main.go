package main

import (
	"context"
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/orchestrator"
	"distributed-data-pipeline-manager/src/parsers"
	"distributed-data-pipeline-manager/src/producer"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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
	setLogLevel(cfg.App.Logger.Level)

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
	messageBytes, err := ParseMessages(data)
	if err != nil {
		log.Fatalf("ERROR: Failed to parse JSON data: %v\n", err)
	}

	// Dynamic topic partition management
	brokers := strings.Join(cfg.App.Kafka.Brokers, ",")
	topic := cfg.App.Kafka.Topics[0]
	consumerGroup := cfg.App.Kafka.ConsumerGroup
	threshold := 10000 // Example threshold
	scaleBy := 2       // Example scale factor
	checkInterval := 10 * time.Second

	// Start the monitoring and scaling handler
	err = ensureTopicPartitions(brokers, topic, 3)
	if err != nil {
		log.Printf("ERROR: Failed to scale partitions for topic %s: %v", topic, err)
	}
	go monitorAndScalePartitions(brokers, topic, consumerGroup, threshold, scaleBy, checkInterval)

	// Initialize Kafka producer
	kafkaProducer, err := producer.NewKafkaProducer(cfg.App.Kafka.Brokers)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize Kafka producer: %v\n", err)
		os.Exit(1)
	}
	defer kafkaProducer.Close()

	// Start producer
	log.Println("DEBUG: Starting producer...")
	err = producer.ProduceMessages(kafkaProducer, cfg.App.Kafka.Topics, parser, messageBytes)
	if err != nil {
		fmt.Printf("ERROR: Failed to produce messages: %v\n", err)
		os.Exit(1)
	}

	// Initialize the pipeline orchestrator
	isTesting := os.Getenv("INTEGRATION_TEST_MODE") == "true"
	timeout := 0 * time.Second
	if isTesting {
		log.Println("INFO: Running in integration test mode")
		timeout = 30 * time.Second // Adjust for integration tests
	}
	orchestrator := orchestrator.NewOrchestrator(cfg, &execute_pipeline.RealCommandExecutor{}, isTesting, timeout)

	// Run the orchestrator
	if err := orchestrator.Run(); err != nil {
		log.Fatalf("ERROR: Orchestrator encountered an issue: %v\n", err)
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
func ParseMessages(data []byte) ([]byte, error) {
	// Try parsing as an object with a "messages" field
	var objectWithMessages struct {
		Messages []string `json:"messages"`
	}
	if err := json.Unmarshal(data, &objectWithMessages); err == nil && len(objectWithMessages.Messages) > 0 {
		return json.Marshal(objectWithMessages.Messages)
	}

	// Try parsing as an array of strings
	var arrayOfStrings []string
	if err := json.Unmarshal(data, &arrayOfStrings); err == nil {
		return json.Marshal(arrayOfStrings)
	}

	// Try parsing as a single object
	var singleMessage map[string]interface{}
	if err := json.Unmarshal(data, &singleMessage); err == nil {
		return json.Marshal([]string{string(data)})
	}

	return nil, fmt.Errorf("unsupported JSON format: %s", string(data))
}

// Monitor and scale kafka partitions
func monitorAndScalePartitions(brokers, topic, group string, threshold, scaleBy int, checkInterval time.Duration) {
	log.Println("INFO: monitorAndScalePartitions started.")
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		// Fetch consumer lag
		consumerLag, err := getConsumerLag(brokers, group, topic)
		if err != nil {
			log.Printf("ERROR: Failed to get consumer lag: %v", err)
			continue
		}

		log.Printf("DEBUG: Consumer lag for topic %s: %d", topic, consumerLag)

		// Check against threshold
		if consumerLag > int64(threshold) {
			log.Printf("INFO: Consumer lag (%d) exceeds threshold (%d). Scaling partitions...", consumerLag, threshold)

			// Create admin client
			adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": brokers})
			if err != nil {
				log.Printf("ERROR: Failed to create admin client: %v", err)
				continue
			}

			// Ensure the admin client is closed properly
			func() {
				defer adminClient.Close()

				metadata, err := adminClient.GetMetadata(&topic, false, 5000)
				if err != nil {
					log.Printf("ERROR: Failed to fetch metadata for topic %s: %v", topic, err)
					return
				}

				currentPartitions := len(metadata.Topics[topic].Partitions)
				newPartitionCount := currentPartitions + scaleBy

				// Scale partitions
				err = ensureTopicPartitions(brokers, topic, newPartitionCount)
				if err != nil {
					log.Printf("ERROR: Failed to scale partitions for topic %s: %v", topic, err)
				} else {
					log.Printf("INFO: Successfully scaled partitions for topic %s to %d", topic, newPartitionCount)
				}
			}()
		}
	}
}

// Get consumer lag top manage kafka topics
func getConsumerLag(brokers, group, topic string) (int64, error) {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers":  brokers,
		"request.timeout.ms": "10000",
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	client, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"group.id":          group,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create kafka client: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	groupPartitions := []kafka.ConsumerGroupTopicPartitions{{
		Group:      group,
		Partitions: []kafka.TopicPartition{{Topic: &topic, Partition: kafka.PartitionAny}},
	}}

	offsetsResult, err := adminClient.ListConsumerGroupOffsets(ctx, groupPartitions)
	if err != nil {
		return 0, fmt.Errorf("failed to list consumer group offsets: %w", err)
	}

	// The result contains the same structure as input, with offsets filled in
	consumerGroupOffsets := offsetsResult.ConsumerGroupsTopicPartitions[0]

	metadata, err := adminClient.GetMetadata(&topic, false, 5000)
	if err != nil {
		return 0, fmt.Errorf("failed to get metadata for topic %s: %w", topic, err)
	}

	var totalLag int64
	for _, partition := range metadata.Topics[topic].Partitions {
		_, high, err := client.GetWatermarkOffsets(topic, partition.ID)
		if err != nil {
			return 0, fmt.Errorf("failed to get watermark offsets for partition %d: %w", partition.ID, err)
		}

		// Find the matching partition in the consumer group offsets
		for _, p := range consumerGroupOffsets.Partitions {
			if p.Partition == partition.ID {
				if p.Offset != kafka.OffsetInvalid && int64(p.Offset) < high {
					totalLag += high - int64(p.Offset)
				}
				break
			}
		}
	}

	return totalLag, nil
}

// Application layer dynamic kafka topic partition management
func ensureTopicPartitions(brokers string, topic string, minPartitions int) error {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": brokers})
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	// Fetch metadata for the topic
	metadata, err := adminClient.GetMetadata(&topic, false, 5000)
	if err != nil {
		return fmt.Errorf("failed to get topic metadata: %w", err)
	}

	// Check the current partition count
	currentPartitions := len(metadata.Topics[topic].Partitions)
	if currentPartitions >= minPartitions {
		log.Printf("Topic %s already has %d partitions (min required: %d)", topic, currentPartitions, minPartitions)
		return nil
	}

	// Add partitions if the current count is less than required
	log.Printf("Increasing partitions for topic %s from %d to %d", topic, currentPartitions, minPartitions)
	results, err := adminClient.CreatePartitions(
		context.Background(),
		[]kafka.PartitionsSpecification{
			{
				Topic:      topic,
				IncreaseTo: minPartitions,
			},
		},
		kafka.SetAdminOperationTimeout(5000),
	)
	if err != nil {
		return fmt.Errorf("failed to increase partitions: %w", err)
	}

	// Log results
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			return fmt.Errorf("partition increase failed for topic %s: %v", topic, result.Error)
		}
	}

	log.Printf("Partitions for topic %s successfully increased to %d", topic, minPartitions)
	return nil
}
