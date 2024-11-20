package consumer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/assert"
)

// MockConsumer is a mock implementation of kafka.Consumer.
type MockConsumer struct {
	messages []kafka.Message
	index    int
	closed   bool
}

// NewMockConsumer initializes a MockConsumer with predefined messages.
func NewMockConsumer(messages []kafka.Message) *MockConsumer {
	return &MockConsumer{
		messages: messages,
		index:    0,
		closed:   false,
	}
}

// ReadMessage returns predefined messages in sequence or an error when exhausted.
func (m *MockConsumer) ReadMessage(timeoutMs int) (*kafka.Message, error) {
	if m.index >= len(m.messages) {
		return nil, fmt.Errorf("no more messages")
	}
	msg := m.messages[m.index]
	m.index++
	return &msg, nil
}

// Close marks the consumer as closed.
func (m *MockConsumer) Close() error {
	m.closed = true
	return nil
}

// TestConsumeMessages validates the behavior of ConsumeMessages.
func TestConsumeMessages(t *testing.T) {
	// Setup mock Kafka messages
	mockMessages := []kafka.Message{
		{Value: []byte("test-message-1")},
		{Value: []byte("test-message-2")},
	}
	mockConsumer := NewMockConsumer(mockMessages)

	// Setup mock Postgres
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Mock the database INSERT operation for both messages
	mock.ExpectExec("INSERT INTO test_table").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO test_table").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Define input parameters
	table := "test_table"
	columns := map[string]string{
		"id":        "id",
		"data":      "data",
		"timestamp": "timestamp",
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Call the function under test
	err = ConsumeMessages(ctx, mockConsumer, db, table, columns)
	assert.NoError(t, err)

	// Assert database expectations
	assert.NoError(t, mock.ExpectationsWereMet())

	// Assert Kafka consumer behavior
	assert.True(t, mockConsumer.closed, "Kafka consumer should be closed after execution")
}
