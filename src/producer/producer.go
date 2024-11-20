package producer

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func ProduceMessages(brokers []string, topics []string, count int) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no brokers provided")
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":       brokers[0],
		"security.protocol":       "PLAINTEXT", // Match Redpanda's configuration
		"api.version.request":     false,       // Disable API version negotiation
		"broker.version.fallback": "2.6.0",     // Fallback to a compatible API version
		"socket.keepalive.enable": true,        // Keep sockets alive
	})

	if err != nil {
		return err
	}
	defer producer.Close()

	// for i := 0; i < count; i++ {
	// 	value := fmt.Sprintf("Message %d", i)
	// 	err := producer.Produce(&kafka.Message{
	// 		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
	// 		Value:          []byte(value),
	// 	}, nil)

	// 	if err != nil {
	// 		fmt.Printf("Failed to produce message: %s\n", err)
	// 	} else {
	// 		fmt.Printf("Produced: %s\n", value)
	// 	}
	// }

	for i := 0; i < count; i++ {
		value := fmt.Sprintf("Message %d", i)
		deliveryChan := make(chan kafka.Event)

		// Produce a message, TODO: Need to be able to handle multiple topics
		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topics[0], Partition: kafka.PartitionAny},
			Value:          []byte(value),
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

	// Flush outstanding messages
	// fmt.Println("Flushing messages...")
	// remaining := producer.Flush(15 * 1000) // Wait up to 15 seconds
	// if remaining > 0 {
	// 	fmt.Printf("WARNING: %d messages were not delivered\n", remaining)
	// } else {
	// 	fmt.Println("All messages delivered.")
	// }

	return nil
}
