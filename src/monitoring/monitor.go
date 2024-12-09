package monitoring

import (
	"context"
	"distributed-data-pipeline-manager/src/producer"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// GetConsumerLagFunc defines a function type for fetching consumer lag.
type GetConsumerLagFunc func(brokers, group, topic string) (int64, error)

// EnsureTopicPartitionsFunc defines a function type for scaling partitions.
type EnsureTopicPartitionsFunc func(brokers, topic string, newPartitionCount int) error

// MonitorConfig holds the configuration for monitoring.
type MonitorConfig struct {
	Brokers       string
	Topic         string
	Group         string
	Threshold     int
	ScaleBy       int
	CheckInterval time.Duration
}

// MonitorAndScale monitors consumer lag and scales partitions when necessary.
func MonitorAndScale(
	cfg MonitorConfig,
	getConsumerLag GetConsumerLagFunc,
	ensureTopicPartitions EnsureTopicPartitionsFunc,
) {
	log.Println("INFO: MonitorAndScale started.")
	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		consumerLag, err := getConsumerLag(cfg.Brokers, cfg.Group, cfg.Topic)
		if err != nil {
			log.Printf("ERROR: Failed to get consumer lag: %v", err)
			continue
		}

		log.Printf("DEBUG: Consumer lag for topic %s: %d", cfg.Topic, consumerLag)

		if consumerLag > int64(cfg.Threshold) {
			log.Printf("INFO: Consumer lag (%d) exceeds threshold (%d). Scaling partitions...", consumerLag, cfg.Threshold)
			if err := ensureTopicPartitions(cfg.Brokers, cfg.Topic, cfg.ScaleBy); err != nil {
				log.Printf("ERROR: Failed to scale partitions: %v", err)
			}
		}
	}
}

// getConsumerLag fetches the consumer lag for the given topic and group.
func GetConsumerLag(brokers, group, topic string) (int64, error) {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers":  brokers,
		"request.timeout.ms": "10000",
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	client, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"group.id":          group,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create kafka client: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	groupPartitions := []kafka.ConsumerGroupTopicPartitions{{
		Group:      group,
		Partitions: []kafka.TopicPartition{{Topic: &topic, Partition: kafka.PartitionAny}},
	}}

	offsetsResult, err := adminClient.ListConsumerGroupOffsets(ctx, groupPartitions)
	if err != nil {
		return 0, fmt.Errorf("failed to list consumer group offsets: %w", err)
	}

	// The result contains the same structure as input, with offsets filled in
	consumerGroupOffsets := offsetsResult.ConsumerGroupsTopicPartitions[0]

	metadata, err := adminClient.GetMetadata(&topic, false, 5000)
	if err != nil {
		return 0, fmt.Errorf("failed to get metadata for topic %s: %w", topic, err)
	}

	var totalLag int64
	for _, partition := range metadata.Topics[topic].Partitions {
		_, high, err := client.GetWatermarkOffsets(topic, partition.ID)
		if err != nil {
			return 0, fmt.Errorf("failed to get watermark offsets for partition %d: %w", partition.ID, err)
		}

		// Find the matching partition in the consumer group offsets
		for _, p := range consumerGroupOffsets.Partitions {
			if p.Partition == partition.ID {
				if p.Offset != kafka.OffsetInvalid && int64(p.Offset) < high {
					totalLag += high - int64(p.Offset)
				}
				break
			}
		}
	}

	return totalLag, nil
}

// scalePartitions scales the number of partitions for the given topic.
func scalePartitions(brokers, topic string, scaleBy int) error {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": brokers})
	if err != nil {
		return err
	}
	defer adminClient.Close()

	metadata, err := adminClient.GetMetadata(&topic, false, 5000)
	if err != nil {
		return err
	}

	currentPartitions := len(metadata.Topics[topic].Partitions)
	newPartitionCount := currentPartitions + scaleBy

	return producer.EnsureTopicPartitions(adminClient, topic, newPartitionCount)
}
