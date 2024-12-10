package producer

import (
	"encoding/json"
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

func (m *MockParser) Parse(data []byte) (interface{}, error) {
	args := m.Called(data)
	return args.Get(0), args.Error(1)
}

func (m *MockParser) Name() string {
	return "mock"
}

func (m *MockParser) Version() string {
	return "1.0.0"
}

// Rest of the mock implementations remain the same
type MockProducer struct {
	ProducedMessages []*kafka.Message
	FlushCount       int
	FlushFunc        func(timeoutMs int) int
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

func TestProduceMessages(t *testing.T) {
	tests := []struct {
		name        string
		mockData    interface{}
		inputData   []byte
		expectError bool
		messages    []string
	}{
		{
			name:        "Simple JSON object",
			mockData:    map[string]interface{}{"name": "John", "age": 30},
			inputData:   []byte(`{"name":"John","age":30}`),
			expectError: false,
			messages:    []string{`{"name":"John","age":30}`},
		},
		{
			name:        "JSON array",
			mockData:    []interface{}{"message1", "message2", "message3"},
			inputData:   []byte(`["message1","message2","message3"]`),
			expectError: false,
			messages:    []string{"message1", "message2", "message3"},
		},
		{
			name:        "Parse error",
			mockData:    nil,
			inputData:   []byte(`invalid json`),
			expectError: true,
			messages:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProducer := &MockProducer{}
			mockParser := &MockParser{}

			if tt.expectError {
				mockParser.On("Parse", tt.inputData).Return(tt.mockData, errors.New("parse error"))
			} else {
				mockParser.On("Parse", tt.inputData).Return(tt.mockData, nil)
			}

			topics := []string{"test-topic"}
			err := ProduceMessages(mockProducer, topics, mockParser, tt.inputData)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.messages), len(mockProducer.ProducedMessages))
				for i, expectedMsg := range tt.messages {
					// Unmarshal both expected and actual JSON for comparison
					var expected, actual map[string]interface{}
					json.Unmarshal([]byte(expectedMsg), &expected)
					json.Unmarshal(mockProducer.ProducedMessages[i].Value, &actual)
					assert.Equal(t, expected, actual)
				}
			}
		})
	}
}

// TestFlushError remains the same
func TestFlushError(t *testing.T) {
	mockProducer := &MockProducer{}
	mockProducer.FlushFunc = func(timeoutMs int) int {
		mockProducer.FlushCount++
		return 1
	}

	mockParser := &MockParser{}
	mockParser.On("Parse", mock.Anything).Return(
		map[string]interface{}{"message": "test"},
		nil,
	)

	topics := []string{"test-topic"}
	data := []byte(`{"message":"test"}`)

	err := ProduceMessages(mockProducer, topics, mockParser, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to flush all messages")
	assert.Equal(t, 3, mockProducer.FlushCount)
}
