package main

import (
	"distributed-data-pipeline-manager/src/bootstrap"
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/orchestrator"
	"distributed-data-pipeline-manager/src/producer"
	"distributed-data-pipeline-manager/src/server"
	"distributed-data-pipeline-manager/src/utils"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// Parse command-line flags
	configPath, port, help := parseFlags()
	if *help {
		showHelpOptions()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := bootstrap.InitializeApp(*configPath, *port)
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	// Set global log level
	utils.SetLogLevel(cfg.App.Logger.Level)

	// Start HTTP server for health checks and metrics
	go server.Start(*port)

	// Enable profiling if configured
	enableProfiling(cfg)

	utils.SetupSignalHandler()

	// Initialize Kafka components
	adminClient, err := bootstrap.InitializeKafka(cfg, bootstrap.NewKafkaAdminClient)
	if err != nil {
		log.Fatalf("ERROR: Kafka initialization failed: %v", err)
	}
	defer adminClient.Close() // Ensure proper cleanup

	// Create Kafka producer
	kafkaProducer, err := producer.NewKafkaProducer(cfg.App.Kafka.Brokers)
	if err != nil {
		log.Fatalf("ERROR: Kafka producer initialization failed: %v", err)
	}

	// Start message production
	startMessageProduction(cfg, kafkaProducer)

	// Start orchestrator for pipeline management
	startOrchestrator(cfg, kafkaProducer)
}

func parseFlags() (*string, *int, *bool) {
	help := flag.Bool("help", false, "Show help for the application")
	configPath := flag.String("config", "", "Path to the configuration file")
	port := flag.Int("port", 8080, "Port for the application to listen on")
	flag.Parse()
	return configPath, port, help
}

func enableProfiling(cfg *config.AppConfig) {
	if cfg.App.Profiling {
		log.Println("INFO: Profiling enabled")
		defer utils.EnableProfiling("cpu.pprof", "mem.pprof")()
	}
}

func startMessageProduction(cfg *config.AppConfig, kafkaProducer *producer.KafkaProducer) {
	defer kafkaProducer.Close()

	parser, err := bootstrap.InitializeParser(cfg.App.Source.Parser, cfg.App.Source.PluginPath)
	if err != nil {
		log.Fatalf("ERROR: Failed to initialize parser: %v", err)
	}

	if err := producer.ProduceMessages(kafkaProducer, cfg.App.Kafka.Topics, parser, nil); err != nil {
		log.Fatalf("ERROR: Failed to produce messages: %v", err)
	}

	log.Println("INFO: Message production completed successfully")
}

func startOrchestrator(cfg *config.AppConfig, kafkaProducer *producer.KafkaProducer) {
	isTesting := os.Getenv("INTEGRATION_TEST_MODE") == "true"
	timeout := time.Duration(0)
	if isTesting {
		log.Println("INFO: Running in integration test mode")
		timeout = 30 * time.Second
	}

	configLoader := config.NewConfigLoader()
	orchestratorInstance := orchestrator.NewOrchestrator(
		configLoader,
		cfg,
		&execute_pipeline.RealCommandExecutor{},
		kafkaProducer,
		isTesting,
		timeout,
	)

	if err := orchestratorInstance.Run(); err != nil {
		log.Fatalf("ERROR: Orchestrator terminated with error: %v", err)
	}

	log.Println("INFO: Orchestrator execution completed successfully")
}

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
