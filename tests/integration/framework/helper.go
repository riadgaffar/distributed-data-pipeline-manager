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
			return nil
		}
	}

	// Read and parse test data file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var messages []interface{}
	if err := json.Unmarshal(data, &messages); err != nil {
		// Try single message if array unmarshal fails
		var singleMessage interface{}
		if err := json.Unmarshal(data, &singleMessage); err != nil {
			return nil
		}
		messages = []interface{}{singleMessage}
	}

	return messages
}

func (h *PipelineTestHelper) ProduceMessagesToKafka(messages []interface{}) error {
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
	for i, msg := range messages {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message %d: %w", i, err)
		}

		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: kafka.PartitionAny,
			},
			Value: msgBytes,
		}, nil)
		if err != nil {
			return fmt.Errorf("failed to produce message %d: %w", i, err)
		}
	}

	// Wait for messages to be delivered
	producer.Flush(15 * 1000)
	return nil
}

func (h *PipelineTestHelper) ValidateProcessedData(expectedCount int) error {
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

func (h *PipelineTestHelper) Setup() error {
	// Start pipeline before running tests
	if err := execute_pipeline.ExecutePipeline(h.configPath, h.executor); err != nil {
		return fmt.Errorf("failed to start pipeline: %w", err)
	}
	return nil
}
