package main

import (
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/monitoring"
	"distributed-data-pipeline-manager/src/orchestrator"
	"distributed-data-pipeline-manager/src/parsers"
	"distributed-data-pipeline-manager/src/producer"
	"distributed-data-pipeline-manager/src/server"
	"distributed-data-pipeline-manager/src/utils"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

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

	// Start HTTP server
	go server.Start(*port)

	log.Println("INFO: Distributed Data Pipeline Manager")
	setLogLevel(cfg.App.Logger.Level)

	// enableProfiling enables CPU and memory profiling
	if cfg.App.Profiling {
		defer utils.EnableProfiling("cpu.pprof", "mem.pprof")()
	}

	utils.SetupSignalHandler()

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
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": brokers})
	if err != nil {
		log.Fatalf("ERROR: Failed to create admin client: %v", err)
	}
	defer adminClient.Close()

	if err := producer.EnsureTopicPartitions(adminClient, topic, 3); err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	// Start monitoring and scaling in a separate goroutine
	monitorConfig := monitoring.MonitorConfig{
		Brokers:       brokers,
		Topic:         topic,
		Group:         consumerGroup,
		Threshold:     threshold,
		ScaleBy:       scaleBy,
		CheckInterval: checkInterval,
	}
	go monitoring.MonitorAndScale(
		monitorConfig,
		func(brokers, group, topic string) (int64, error) {
			return monitoring.GetConsumerLag(brokers, group, topic)
		},
		func(brokers, topic string, newPartitionCount int) error {
			adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": brokers})
			if err != nil {
				return fmt.Errorf("failed to create Kafka admin client: %w", err)
			}
			defer adminClient.Close()
			return producer.EnsureTopicPartitions(adminClient, topic, newPartitionCount)
		},
	)

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

	// Run orchestrator
	if err := orchestrator.Run(); err != nil {
		log.Fatalf("ERROR: Orchestrator terminated with error: %v\n", err)
	}

	log.Println("INFO: Application shutdown complete.")

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
