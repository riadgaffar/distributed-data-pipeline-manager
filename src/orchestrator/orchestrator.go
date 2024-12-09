package orchestrator

import (
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"distributed-data-pipeline-manager/src/producer"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Orchestrator manages the lifecycle of the pipeline
type ConfigLoader interface {
	LoadConfig(string) (*config.AppConfig, error)
}

type Orchestrator struct {
	configLoader ConfigLoader
	config       *config.AppConfig
	executor     execute_pipeline.CommandExecutor
	producer     producer.Producer
	stopChan     chan os.Signal
	timeout      time.Duration
	isTesting    bool
}

// NewOrchestrator creates a new pipeline orchestrator
func NewOrchestrator(configLoader ConfigLoader, cfg *config.AppConfig, executor execute_pipeline.CommandExecutor, producer producer.Producer, isTesting bool, timeout time.Duration) *Orchestrator {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	return &Orchestrator{
		configLoader: configLoader,
		config:       cfg,
		executor:     executor,
		producer:     producer,
		stopChan:     stopChan,
		timeout:      timeout,
		isTesting:    isTesting,
	}
}

// Run starts the pipeline execution
func (o *Orchestrator) Run() error {
	log.Println("INFO: Starting pipeline orchestrator")

	defer o.producer.Close()

	// Execute pipeline
	log.Println("DEBUG: Executing pipeline...")
	go func() {
		err := execute_pipeline.ExecutePipeline(os.Getenv("CONFIG_PATH"), o.executor)
		if err != nil {
			log.Printf("ERROR: Pipeline execution failed: %v\n", err)
			os.Exit(1)
		}
	}()

	// Monitor for graceful shutdown
	return o.monitor()
}

// monitor waits for shutdown signals or testing timeouts
func (o *Orchestrator) monitor() error {
	if o.isTesting && o.timeout > 0 {
		log.Printf("INFO: Running in testing mode with a timeout of %v\n", o.timeout)
		select {
		case <-time.After(o.timeout):
			log.Println("INFO: Timeout reached, stopping pipeline...")
			return o.stopPipeline()
		case <-o.stopChan:
			log.Println("INFO: Received termination signal, stopping pipeline...")
			return o.stopPipeline()
		}
	} else {
		log.Println("INFO: Running in production mode. Press Ctrl+C to stop.")
		<-o.stopChan
		log.Println("INFO: Received termination signal, stopping pipeline...")
		return o.stopPipeline()
	}
}

// stopPipeline handles graceful shutdown
func (o *Orchestrator) stopPipeline() error {
	log.Println("DEBUG: Stopping pipeline...")
	return o.executor.StopPipeline()
}
