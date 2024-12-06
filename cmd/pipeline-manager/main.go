package main

import (
	"context"
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/orchestrator"
	"distributed-data-pipeline-manager/src/parsers"
	"distributed-data-pipeline-manager/src/producer"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

var cpuProfileFile *os.File

// Rearranged Kafka Operations and Message Handling

func main() {
	// Define and parse flags
	help := flag.Bool("help", false, "Show help for the application")
	configPath := flag.String("config", "", "Path to the configuration file")
	port := flag.Int("port", 8080, "Port for the application to listen on")
	flag.Parse()

	if *help {
		showHelpOptions()
		os.Exit(0)
	}

	// If configPath is empty, handle it (e.g., log or set a fallback)
	if *configPath == "" {
		log.Println("INFO: --config flag not provided, configPath is empty.")
	} else {
		if err := validateConfigPath(*configPath); err != nil {
			log.Fatalf("Invalid configuration path: %v\n", err)
		}
	}

	if err := validatePort(*port); err != nil {
		log.Fatalf("Invalid port: %v\n", err)
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	log.Printf("INFO: Starting pipeline manager on port %d with config: %s\n", *port, *configPath)
	startServer(*port)

	log.Println("INFO: Distributed Data Pipeline Manager")
	setLogLevel(cfg.App.Logger.Level)

	if cfg.App.Profiling {
		defer enableProfiling()()
	}

	setupSignalHandler()

	// Initialize the parser
	parser, err := initializeParser(cfg.App.Source.Parser)
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	// Kafka Brokers and Topic Configuration
	brokers := strings.Join(cfg.App.Kafka.Brokers, ",")
	topic := strings.Join(cfg.App.Kafka.Topics, ",")
	consumerGroup := cfg.App.Kafka.ConsumerGroup
	threshold := 10000 // threshold for partition scaling
	scaleBy := 2       // scale factor
	checkInterval := 10 * time.Second

	// Ensure Minimum Partitions for Topic
	err = ensureTopicPartitions(brokers, topic, 3)
	if err != nil {
		log.Printf("ERROR: Failed to scale partitions for topic %s: %v", topic, err)
	}

	// Start Monitoring and Scaling for Topic
	go monitorAndScalePartitions(brokers, topic, consumerGroup, threshold, scaleBy, checkInterval)

	// Kafka Producer Initialization
	kafkaProducer, err := producer.NewKafkaProducer(cfg.App.Kafka.Brokers)
	if err != nil {
		log.Fatalf("ERROR: Failed to initialize Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	log.Println("DEBUG: Kafka producer initialized and ready.")

	// Produce Messages Dynamically (replace `messageBytes` with dynamic streaming)
	log.Println("DEBUG: Starting message production...")
	err = producer.ProduceMessages(kafkaProducer, cfg.App.Kafka.Topics, parser, nil) // Adjust args if needed
	if err != nil {
		log.Fatalf("ERROR: Failed to produce messages: %v\n", err)
	}

	// Pipeline Orchestration
	isTesting := os.Getenv("INTEGRATION_TEST_MODE") == "true"
	timeout := 0 * time.Second
	if isTesting {
		log.Println("INFO: Running in integration test mode")
		timeout = 30 * time.Second
	}

	orchestrator := orchestrator.NewOrchestrator(cfg, &execute_pipeline.RealCommandExecutor{}, isTesting, timeout)
	if err := orchestrator.Run(); err != nil {
		log.Fatalf("ERROR: Orchestrator encountered an issue: %v\n", err)
	}

	log.Println("INFO: Pipeline executed successfully.")
}

// showHelpOptions displays help information
func showHelpOptions() {
	fmt.Println("Pipeline Manager Application")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  pipeline_manager [options]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
}

// validateConfigPath checks if the configuration file path exists
func validateConfigPath(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", configPath)
	}
	return nil
}

// validatePort ensures the provided port is within a valid range
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number %d is out of range (1-65535)", port)
	}
	return nil
}

// startServer starts the HTTP server on the specified port
func startServer(port int) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	go func() {
		if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()
}

// loadConfig loads the configuration from the CONFIG_PATH environment variable or the --config flag
func loadConfig(configPathFlag string) (*config.AppConfig, error) {
	// Determine configuration path precedence
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = configPathFlag
		log.Printf("INFO: CONFIG_PATH not set, using --config flag value: %s", configPath)
	} else {
		log.Printf("INFO: Using CONFIG_PATH environment variable: %s", configPath)
	}

	// Validate the configuration path
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file does not exist: %s", configPath)
	}

	// Load the configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from %s: %w", configPath, err)
	}

	log.Printf("INFO: Loaded configuration: %+v", cfg)
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
