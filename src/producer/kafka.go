package producer

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// AdminClient defines the interface for Kafka admin operations.
type AdminClient interface {
	GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
	CreatePartitions(ctx context.Context, partitionsSpec []kafka.PartitionsSpecification, opts ...kafka.CreatePartitionsAdminOption) ([]kafka.TopicResult, error)
	Close()
}

// EnsureTopicPartitions ensures the Kafka topic has at least the specified number of partitions.
func EnsureTopicPartitions(adminClient AdminClient, topic string, minPartitions int) error {
	metadata, err := adminClient.GetMetadata(&topic, false, 5000)
	if err != nil {
		return fmt.Errorf("failed to get topic metadata: %w", err)
	}

	currentPartitions := len(metadata.Topics[topic].Partitions)
	if currentPartitions >= minPartitions {
		return nil
	}

	_, err = adminClient.CreatePartitions(context.Background(), []kafka.PartitionsSpecification{
		{Topic: topic, IncreaseTo: minPartitions},
	})
	if err != nil {
		return fmt.Errorf("failed to increase partitions: %w", err)
	}

	return nil
}
