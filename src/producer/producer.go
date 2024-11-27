package producer

import (
	"fmt"

	"distributed-data-pipeline-manager/src/parsers"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// Producer defines an interface for producing messages.
type Producer interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
}

// KafkaProducer wraps the confluent-kafka-go producer to implement the Producer interface.
type KafkaProducer struct {
	producer *kafka.Producer
}

// NewKafkaProducer creates a new KafkaProducer.
func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("no brokers provided")
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":       brokers[0],
		"security.protocol":       "PLAINTEXT",
		"api.version.request":     false,
		"broker.version.fallback": "2.6.0",
		"socket.keepalive.enable": true,
	})
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{producer: producer}, nil
}

// Produce sends a message to Kafka.
func (kp *KafkaProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return kp.producer.Produce(msg, deliveryChan)
}

// Close shuts down the Kafka producer.
func (kp *KafkaProducer) Close() {
	kp.producer.Close()
}

// ProduceMessages sends messages using the provided producer and parser.
func ProduceMessages(producer Producer, topics []string, count int, parser parsers.Parser, data []byte) error {
	// Step 1: Parse the data
	messages, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse data: %w", err)
	}

	// Step 2: Produce each message
	for i, message := range messages {
		if i >= count {
			break
		}

		deliveryChan := make(chan kafka.Event)
		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topics[0], Partition: kafka.PartitionAny},
			Value:          []byte(message),
		}, deliveryChan)

		if err != nil {
			return fmt.Errorf("failed to produce message: %w", err)
		}

		e := <-deliveryChan
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("delivery error: %w", m.TopicPartition.Error)
		}
		close(deliveryChan)
	}

	return nil
}
