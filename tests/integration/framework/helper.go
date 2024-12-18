package framework

import (
	"database/sql"
	"distributed-data-pipeline-manager/src/config"
	"distributed-data-pipeline-manager/src/execute_pipeline"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type PipelineTestHelper struct {
	configPath string
	config     *config.AppConfig
	executor   execute_pipeline.CommandExecutor
}

func NewPipelineTestHelper(configPath string) *PipelineTestHelper {
	return &PipelineTestHelper{
		configPath: configPath,
		executor:   &execute_pipeline.RealCommandExecutor{},
	}
}

func (h *PipelineTestHelper) ParseJSONTestMessages(path string) []interface{} {
	// Load config if not already loaded
	if h.config == nil {
		var err error
		h.config, err = config.LoadConfig(h.configPath)
		if err != nil {
			log.Printf("Config load failed: %v", err)
			return nil
		}
	}

	// Read and parse test data file
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("File read failed: %v", err)
		return nil
	}

	var wrapper struct {
		Messages []interface{} `json:"messages"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		log.Printf("JSON unmarshal failed: %v", err)
		return nil
	}

	log.Printf("Parsed messages: %+v", wrapper.Messages)
	return wrapper.Messages
}

func (h *PipelineTestHelper) ProduceMessagesToKafka(messages []interface{}) error {
	log.Printf("Attempting to produce %d messages to topic %s", len(messages), h.config.App.Kafka.Topics[0])

	if h.config == nil {
		var err error
		h.config, err = config.LoadConfig(h.configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(h.config.App.Kafka.Brokers, ","),
	})
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}
	defer producer.Close()

	// Use first topic from config
	topic := h.config.App.Kafka.Topics[0]

	// Produce messages
	deliveryChan := make(chan kafka.Event)
	for i, msg := range messages {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message %d: %w", i, err)
		}

		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          msgBytes,
		}, deliveryChan)

		e := <-deliveryChan
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			log.Printf("Delivery failed: %v", m.TopicPartition.Error)
		} else {
			log.Printf("Delivered message to topic %s [%d] at offset %v",
				*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
		}
	}

	// Add delay after producing messages
	log.Printf("Waiting for messages to be processed...")
	time.Sleep(2 * time.Second)

	return nil
}

func (h *PipelineTestHelper) Setup() error {
	log.Printf("Starting pipeline setup...")

	if err := execute_pipeline.ExecutePipeline(h.configPath, h.executor); err != nil {
		return fmt.Errorf("failed to start pipeline: %w", err)
	}

	// Verify pipeline is running
	if err := h.waitForPipeline(); err != nil {
		return fmt.Errorf("pipeline failed to start: %w", err)
	}

	return nil
}

func (h *PipelineTestHelper) waitForPipeline() error {
	timeout := time.After(30 * time.Second)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for pipeline")
		case <-tick:
			// Check if pipeline is processing by attempting to validate
			if err := h.checkRecordCount(0); err == nil {
				log.Printf("Pipeline is ready - database connection verified")
				return nil
			}
		}
	}
}

func (h *PipelineTestHelper) waitForKafka() error {
	timeout := time.After(30 * time.Second)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for Kafka")
		case <-tick:
			// Try to connect to Kafka
			if err := h.checkKafkaConnection(); err == nil {
				return nil
			}
		}
	}
}

func (h *PipelineTestHelper) checkKafkaConnection() error {
	// Implement Kafka connection check here
	// For example:
	config := &kafka.ConfigMap{"bootstrap.servers": h.config.App.Kafka.Brokers}
	producer, err := kafka.NewProducer(config)
	if err != nil {
		return err
	}
	defer producer.Close()
	return nil
}

func (h *PipelineTestHelper) ValidateProcessedData(expectedCount int) error {
	// Retry with backoff until records appear or timeout
	return retry.Do(
		func() error {
			return h.checkRecordCount(expectedCount)
		},
		retry.Attempts(5),
		retry.Delay(1*time.Second),
	)
}

func (h *PipelineTestHelper) checkRecordCount(expectedCount int) error {
	if h.config == nil {
		var err error
		h.config, err = config.LoadConfig(h.configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	db, err := sql.Open("postgres", h.config.App.Postgres.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Add debug logging
	log.Printf("Checking table: %s", h.config.App.Postgres.Table)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", h.config.App.Postgres.Table)
	var count int
	err = db.QueryRow(query).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query count: %w", err)
	}

	log.Printf("Found %d records, expected %d", count, expectedCount)

	if count != expectedCount {
		// Query actual data for debugging
		rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", h.config.App.Postgres.Table))
		if err != nil {
			log.Printf("Failed to query data: %v", err)
		} else {
			defer rows.Close()
			cols, _ := rows.Columns()
			log.Printf("Table columns: %v", cols)
		}

		return fmt.Errorf("expected %d records, got %d", expectedCount, count)
	}

	return nil
}
func (h *PipelineTestHelper) StopPipeline() error {
	if h.executor == nil {
		return fmt.Errorf("no pipeline process to stop")
	}

	if err := h.executor.StopPipeline(); err != nil {
		return fmt.Errorf("failed to stop pipeline: %w", err)
	}
	return nil
}
