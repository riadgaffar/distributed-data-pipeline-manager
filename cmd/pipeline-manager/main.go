package main

import (
	"distributed-data-pipeline-manager/src/bootstrap"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/orchestrator"
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

	// Load configuration
	cfg, err := bootstrap.InitializeApp(*configPath, *port)
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	// Set log level
	utils.SetLogLevel(cfg.App.Logger.Level)

	log.Printf("INFO: Starting pipeline manager on port %d with config: %s\n", *port, *configPath)

	// Start HTTP server
	go server.Start(*port)

	// enableProfiling enables CPU and memory profiling
	if cfg.App.Profiling {
		defer utils.EnableProfiling("cpu.pprof", "mem.pprof")()
	}

	utils.SetupSignalHandler()

	// Initialize the parser
	parser, err := bootstrap.InitializeParser(cfg.App.Source.Parser)
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	// Kafka Brokers and Topic Configuration
	brokers := strings.Join(cfg.App.Kafka.Brokers, ",")
	topic := strings.Join(cfg.App.Kafka.Topics, ",")

	// Ensure Minimum Partitions for Topic
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": brokers})
	if err != nil {
		log.Fatalf("ERROR: Failed to create admin client: %v", err)
	}
	defer adminClient.Close()

	if err := producer.EnsureTopicPartitions(adminClient, topic, 3); err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	// Use the InitializeKafka function from bootstrap.go
	if err := bootstrap.InitializeKafka(cfg, bootstrap.NewKafkaAdminClient); err != nil {
		log.Fatalf("ERROR: Kafka initialization failed: %v\n", err)
	}

	// Kafka Producer Initialization
	kafkaProducer, err := producer.NewKafkaProducer(cfg.App.Kafka.Brokers)
	if err != nil {
		log.Fatalf("ERROR: Failed to initialize Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	log.Println("DEBUG: Kafka producer initialized and ready.")

	// Produce Messages Dynamically (replace `messageBytes` with dynamic streaming)
	log.Println("DEBUG: Starting message production...")
	err = producer.ProduceMessages(kafkaProducer, cfg.App.Kafka.Topics, parser, nil)
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
