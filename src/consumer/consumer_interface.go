package consumer

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaConsumer interface {
	ReadMessage(timeoutMs int) (*kafka.Message, error)
	Close() error
}

type RealKafkaConsumer struct {
	consumer *kafka.Consumer
}

func (r *RealKafkaConsumer) ReadMessage(timeoutMs int) (*kafka.Message, error) {
	return r.consumer.ReadMessage(time.Duration(timeoutMs) * time.Millisecond)
}

func (r *RealKafkaConsumer) Close() error {
	return r.consumer.Close()
}
