package integration

import (
	"database/sql"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Produce test messages to Kafka/Redpanda
func produceTestMessages(brokers []string, topic string, messages []string) error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokers[0]})
	if err != nil {
		return err
	}
	defer producer.Close()

	for _, message := range messages {
		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: kafka.PartitionAny,
			},
			Value: []byte(message),
		}, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// Fetch processed data from Postgres
func fetchProcessedData(url, table string) ([]ProcessedData, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProcessedData
	for rows.Next() {
		var data ProcessedData
		err := rows.Scan(&data.Data)
		if err != nil {
			return nil, err
		}
		results = append(results, data)
	}

	return results, nil
}

type ProcessedData struct {
	Data string
}
