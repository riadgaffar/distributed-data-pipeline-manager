package producer

import (
	"fmt"
	"log"
	"math/rand"

	"strings"

	"distributed-data-pipeline-manager/src/parsers"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Producer defines an interface for producing messages.
type Producer interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
	Flush(timeoutMs int) int
}

// KafkaProducer wraps the confluent-kafka-go producer to implement the Producer interface.
type KafkaProducer struct {
	producer *kafka.Producer
}

// NewKafkaProducer creates a new KafkaProducer with support for multiple brokers.
func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("no brokers provided")
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"acks":                         "all",
		"bootstrap.servers":            strings.Join(brokers, ","),
		"security.protocol":            "PLAINTEXT",
		"batch.size":                   65536,  // 64 KB batch size for better throughput
		"linger.ms":                    200,    // Allow more time to build larger batches
		"compression.type":             "lz4",  // Fast compression
		"queue.buffering.max.messages": 200000, // Increase buffer size for high message volume
		"socket.keepalive.enable":      true,
		"message.send.max.retries":     10,   // Increase retries for reliability
		"retry.backoff.ms":             300,  // Slightly longer backoff between retries
		"enable.idempotence":           true, // Exactly-once delivery
	})

	if err != nil {
		return nil, err
	}

	return &KafkaProducer{producer: producer}, nil
}

// Produce sends a single message to Kafka.
func (kp *KafkaProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return kp.producer.Produce(msg, deliveryChan)
}

// Flush ensures all outstanding messages are delivered before shutting down.
func (kp *KafkaProducer) Flush(timeoutMs int) int {
	return kp.producer.Flush(timeoutMs)
}

// Close shuts down the Kafka producer.
func (kp *KafkaProducer) Close() {
	kp.producer.Close()
}

// ProduceMessages sends multiple messages using the provided producer and parser.
func ProduceMessages(producer Producer, topics []string, parser parsers.Parser, data []byte) error {
	// Step 1: Parse the data
	messages, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse data: %w", err)
	}

	// Create a delivery channel
	deliveryChan := make(chan kafka.Event, len(messages))

	// Step 2: Produce messages with batching
	for _, message := range messages {
		key := fmt.Sprintf("key-%d", rand.Intn(1000000)) // Generate a unique key for each message
		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topics[0], Partition: kafka.PartitionAny},
			Key:            []byte(key), // Add a key for partitioning
			Value:          []byte(message),
		}, deliveryChan)
		if err != nil {
			return fmt.Errorf("failed to produce message: %w", err)
		}
	}

	// Wait for delivery confirmations
	for i := 0; i < len(messages); i++ {
		event := <-deliveryChan
		msg := event.(*kafka.Message)
		if msg.TopicPartition.Error != nil {
			return fmt.Errorf("failed to deliver message: %w", msg.TopicPartition.Error)
		}
	}

	close(deliveryChan)

	// Step 3: Flush all messages with retry logic
	maxFlushRetries := 3
	for retry := 0; retry < maxFlushRetries; retry++ {
		unflushed := producer.Flush(30000) // 30 seconds timeout
		if unflushed == 0 {
			break // Successfully flushed all messages
		}

		log.Printf("Retrying flush (%d/%d): %d messages still in queue", retry+1, maxFlushRetries, unflushed)

		if retry == maxFlushRetries-1 {
			return fmt.Errorf("failed to flush all messages after %d retries, %d messages still in queue", maxFlushRetries, unflushed)
		}
	}

	return nil
}
