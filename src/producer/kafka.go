package producer

import (
	"context"
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// AdminClient defines the interface for Kafka admin operations, enabling dependency injection.
type AdminClient interface {
	GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
	CreatePartitions(ctx context.Context, partitionsSpec []kafka.PartitionsSpecification, opts ...kafka.CreatePartitionsAdminOption) ([]kafka.TopicResult, error)
	Close()
}

// EnsureTopicPartitions ensures that a Kafka topic has the minimum required partitions.
func EnsureTopicPartitions(adminClient AdminClient, topic string, minPartitions int) error {
	// Fetch metadata for the given topic
	metadata, err := adminClient.GetMetadata(&topic, false, 5000)
	if err != nil {
		return fmt.Errorf("failed to get topic metadata: %w", err)
	}

	// Check the current number of partitions
	currentPartitions := len(metadata.Topics[topic].Partitions)
	if currentPartitions >= minPartitions {
		log.Printf("INFO: Topic %s already has %d partitions", topic, currentPartitions)
		return nil
	}

	// Create partitions to meet the minimum requirement
	results, err := adminClient.CreatePartitions(
		context.Background(),
		[]kafka.PartitionsSpecification{{
			Topic:      topic,
			IncreaseTo: minPartitions,
		}},
		kafka.SetAdminOperationTimeout(5000),
	)
	if err != nil {
		return fmt.Errorf("failed to increase partitions: %w", err)
	}

	// Check the results for errors
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			return fmt.Errorf("partition increase failed for topic %s: %v", topic, result.Error)
		}
	}

	log.Printf("INFO: Partitions for topic %s successfully increased to %d", topic, minPartitions)
	return nil
}
