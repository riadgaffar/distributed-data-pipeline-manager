package producer

import (
	"errors"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockParser simulates the parsers.Parser interface
type MockParser struct {
	mock.Mock
}

func (m *MockParser) Parse(data []byte) ([]string, error) {
	args := m.Called(data)
	return args.Get(0).([]string), args.Error(1)
}

// MockProducer mocks the Producer interface for testing.
type MockProducer struct {
	ProducedMessages []*kafka.Message
	FlushCount       int
	FlushFunc        func(timeoutMs int) int // Renamed to FlushFunc to avoid naming conflict
}

func (mp *MockProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	mp.ProducedMessages = append(mp.ProducedMessages, msg)
	if deliveryChan != nil {
		deliveryChan <- &kafka.Message{TopicPartition: kafka.TopicPartition{Topic: msg.TopicPartition.Topic}}
	}
	return nil
}

func (mp *MockProducer) Flush(timeoutMs int) int {
	if mp.FlushFunc != nil {
		return mp.FlushFunc(timeoutMs)
	}
	mp.FlushCount++
	return 0
}

func (mp *MockProducer) Close() {}

// TestNewKafkaProducer tests the creation of a KafkaProducer with multiple brokers.
func TestNewKafkaProducer(t *testing.T) {
	brokers := []string{"broker1:9092", "broker2:9092"}
	producer, err := NewKafkaProducer(brokers)

	assert.NoError(t, err, "Expected no error when creating a KafkaProducer")
	assert.NotNil(t, producer, "Expected a non-nil KafkaProducer instance")

	defer producer.Close()
}

// TestProduceMessages tests message production with multiple brokers and batching.
func TestProduceMessages(t *testing.T) {
	mockProducer := &MockProducer{}
	mockParser := &MockParser{}
	mockParser.On("Parse", mock.Anything).Return([]string{"message1", "message2", "message3"}, nil)

	topics := []string{"test-topic"}
	data := []byte("test-data") // Fixed: Properly closed the byte array

	err := ProduceMessages(mockProducer, topics, mockParser, data)
	assert.NoError(t, err, "Expected no error when producing messages")

	// Validate that all parsed messages were produced
	assert.Equal(t, 3, len(mockProducer.ProducedMessages), "Expected all messages to be produced")
	assert.Equal(t, "message1", string(mockProducer.ProducedMessages[0].Value), "Message content mismatch")
	assert.Equal(t, "message2", string(mockProducer.ProducedMessages[1].Value), "Message content mismatch")
	assert.Equal(t, "message3", string(mockProducer.ProducedMessages[2].Value), "Message content mismatch")

	// Validate that Flush was called
	assert.Equal(t, 1, mockProducer.FlushCount, "Flush should have been called once")
}

// TestProduceMessages_Failure tests error handling during message production.
func TestProduceMessages_Failure(t *testing.T) {
	mockProducer := &MockProducer{}
	mockParser := &MockParser{}
	mockParser.On("Parse", mock.Anything).Return([]string{}, errors.New("parse error"))

	topics := []string{"test-topic"}
	data := []byte("test-data") // Fixed: Properly closed the byte array

	err := ProduceMessages(mockProducer, topics, mockParser, data)
	assert.Error(t, err, "Expected error when parser fails")
	assert.Contains(t, err.Error(), "failed to parse data", "Expected parse error in the error message")
}

// TestFlushError simulates an error during flush and validates the behavior.
func TestFlushError(t *testing.T) {
	mockProducer := &MockProducer{}
	mockProducer.FlushFunc = func(timeoutMs int) int {
		mockProducer.FlushCount++
		return 1 // Simulate one unflushed message
	}

	mockParser := &MockParser{}
	mockParser.On("Parse", mock.Anything).Return([]string{"message1", "message2"}, nil)

	topics := []string{"test-topic"}
	data := []byte("test-data")

	err := ProduceMessages(mockProducer, topics, mockParser, data)
	assert.Error(t, err, "Expected error due to unflushed messages")
	assert.Contains(t, err.Error(), "failed to flush all messages", "Expected flush error in the error message")

	// Ensure FlushCount matches retries in ProduceMessages
	retryCount := 3 // Match this to the actual retry logic in ProduceMessages
	assert.Equal(t, retryCount, mockProducer.FlushCount, "Flush should have been called the expected number of times")
}
