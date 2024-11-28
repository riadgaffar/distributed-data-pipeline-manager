package integration

import (
	"database/sql"
	"distributed-data-pipeline-manager/src/parsers"
	"distributed-data-pipeline-manager/src/producer"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type ProcessedData struct {
	ID        string    // UUID
	Timestamp time.Time // Timestamp
	Data      string    // Actual processed data
}

// Produce test messages to Kafka/Redpanda
// Helper function to produce test messages to Kafka
func produceTestMessages(brokers []string, topics []string, messageCount int, parser parsers.Parser, data []byte) error {
	// Initialize Kafka producer
	kafkaProducer, err := producer.NewKafkaProducer(brokers)
	if err != nil {
		return fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}
	defer kafkaProducer.Close()

	// Start producer
	log.Println("DEBUG: Starting producer...")
	err = producer.ProduceMessages(kafkaProducer, topics, messageCount, parser, data)
	if err != nil {
		fmt.Printf("ERROR: Failed to produce messages: %v\n", err)
		os.Exit(1)
	}
	return nil
}

// Fetch processed data from Postgres
func fetchProcessedData(url, table string) ([]ProcessedData, error) {
	// Register the Postgres driver with database/sql
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT id, timestamp, data FROM %s", table)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProcessedData
	for rows.Next() {
		var data ProcessedData
		err := rows.Scan(&data.ID, &data.Timestamp, &data.Data)
		if err != nil {
			return nil, err
		}
		results = append(results, data)
	}

	return results, nil
}
