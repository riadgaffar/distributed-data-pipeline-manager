package producer

import (
	"errors"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/kafka"
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

// MockProducer simulates the Producer interface
type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	args := m.Called(msg, deliveryChan)

	// Simulate asynchronous delivery report
	if deliveryChan != nil {
		go func() {
			defer func() {
				// Avoid panic if the channel is closed
				recover()
			}()
			// Send a simulated delivery result only if no error is expected
			if args.Error(0) == nil {
				deliveryChan <- &kafka.Message{
					TopicPartition: kafka.TopicPartition{
						Topic:     msg.TopicPartition.Topic,
						Partition: 0,
						Error:     nil, // Simulate success
					},
				}
			}
		}()
	}
	return args.Error(0)
}

func (m *MockProducer) Close() {
	m.Called()
}

// TestProduceMessages_Success validates successful message production
func TestProduceMessages_Success(t *testing.T) {
	mockProducer := new(MockProducer)
	mockParser := new(MockParser)

	testData := []byte(`test data`)
	parsedMessages := []string{"message1", "message2"}
	topics := []string{"test-topic"}
	messageCount := 2

	// Mock behaviors
	mockParser.On("Parse", testData).Return(parsedMessages, nil)
	mockProducer.On("Produce", mock.Anything, mock.Anything).Return(nil)

	// Execute
	err := ProduceMessages(mockProducer, topics, messageCount, mockParser, testData)

	// Assert
	assert.NoError(t, err)
	mockParser.AssertExpectations(t)
	mockProducer.AssertNumberOfCalls(t, "Produce", 2)
}

// TestNewKafkaProducer_EmptyBrokers checks the error for empty brokers
func TestNewKafkaProducer_EmptyBrokers(t *testing.T) {
	producer, err := NewKafkaProducer([]string{})

	assert.Error(t, err)
	assert.Nil(t, producer)
	assert.Contains(t, err.Error(), "no brokers provided")
}

// TestProduceMessages_ParserError simulates a parsing failure
func TestProduceMessages_ParserError(t *testing.T) {
	mockProducer := new(MockProducer)
	mockParser := new(MockParser)

	testData := []byte(`invalid data`)
	mockParser.On("Parse", testData).Return([]string{}, errors.New("mock parser error"))

	// Execute
	err := ProduceMessages(mockProducer, []string{"test-topic"}, 1, mockParser, testData)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock parser error")
	mockParser.AssertExpectations(t)
	mockProducer.AssertNotCalled(t, "Produce")
}

// TestProduceMessages_EmptyMessages verifies behavior with empty messages
func TestProduceMessages_EmptyMessages(t *testing.T) {
	mockProducer := new(MockProducer)
	mockParser := new(MockParser)

	testData := []byte(`[]`) // No data
	mockParser.On("Parse", testData).Return([]string{}, nil)

	// Execute
	err := ProduceMessages(mockProducer, []string{"test-topic"}, 0, mockParser, testData)

	// Assert
	assert.NoError(t, err)
	mockParser.AssertExpectations(t)
	mockProducer.AssertNotCalled(t, "Produce")
}

func TestProduceMessages_ProducerError(t *testing.T) {
	mockProducer := new(MockProducer)
	mockParser := new(MockParser)

	testData := []byte(`{"message": "test payload"}`)
	parsedMessages := []string{`{"message": "test payload"}`}
	topics := []string{"error-topic"}
	messageCount := 1

	// Setup mock parser
	mockParser.On("Parse", testData).Return(parsedMessages, nil)

	// Setup mock producer with direct error return
	mockProducer.On("Produce", mock.MatchedBy(func(msg *kafka.Message) bool {
		// Initialize the topic to prevent nil pointer
		topic := topics[0]
		msg.TopicPartition.Topic = &topic
		return true
	}), mock.Anything).Return(errors.New("kafka broker connection failed"))

	// Execute
	err := ProduceMessages(mockProducer, topics, messageCount, mockParser, testData)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kafka broker connection failed")
	mockParser.AssertExpectations(t)
	mockProducer.AssertNumberOfCalls(t, "Produce", 1)
}
