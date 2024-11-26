package producer

import (
	"fmt"

	"distributed-data-pipeline-manager/src/parsers"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func ProduceMessages(brokers []string, topics []string, count int, parser parsers.Parser, data []byte) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no brokers provided")
	}

	// Step 1: Parse the data using the provided parser
	messages, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse data: %v", err)
	}

	// Step 2: Initialize the producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":       brokers[0],
		"security.protocol":       "PLAINTEXT",
		"api.version.request":     false,
		"broker.version.fallback": "2.6.0",
		"socket.keepalive.enable": true,
	})
	if err != nil {
		return err
	}
	defer producer.Close()

	// Step 3: Produce parsed messages
	for i, message := range messages {
		if i >= count { // Limit by count
			break
		}

		deliveryChan := make(chan kafka.Event)
		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topics[0], Partition: kafka.PartitionAny},
			Value:          []byte(message),
		}, deliveryChan)

		if err != nil {
			fmt.Printf("Failed to produce message: %s\n", err)
			close(deliveryChan)
			continue
		}

		// Wait for delivery report
		e := <-deliveryChan
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
		}
		close(deliveryChan)
	}

	return nil
}
