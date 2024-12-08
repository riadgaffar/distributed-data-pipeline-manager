package producer

import (
	"context"
	"fmt"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAdminClient is a mock implementation of AdminClient.
type MockAdminClient struct {
	mock.Mock
}

func (m *MockAdminClient) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
	args := m.Called(topic, allTopics, timeoutMs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*kafka.Metadata), args.Error(1)
}

func (m *MockAdminClient) CreatePartitions(ctx context.Context, partitionsSpec []kafka.PartitionsSpecification, opts ...kafka.CreatePartitionsAdminOption) ([]kafka.TopicResult, error) {
	args := m.Called(ctx, partitionsSpec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]kafka.TopicResult), args.Error(1)
}

func (m *MockAdminClient) Close() {
	m.Called()
}

func TestEnsureTopicPartitions(t *testing.T) {
	tests := []struct {
		name          string
		topic         string
		minPartitions int
		setupMock     func(*MockAdminClient)
		wantErr       bool
		errorMsg      string
	}{
		{
			name:          "successful partition increase",
			topic:         "test-topic",
			minPartitions: 3,
			setupMock: func(m *MockAdminClient) {
				m.On("GetMetadata", mock.AnythingOfType("*string"), false, 5000).Return(
					&kafka.Metadata{
						Topics: map[string]kafka.TopicMetadata{
							"test-topic": {Partitions: []kafka.PartitionMetadata{{}, {}}},
						},
					},
					nil,
				)
				m.On("CreatePartitions", mock.Anything, mock.Anything).Return([]kafka.TopicResult{{}}, nil)
			},
			wantErr: false,
		},
		{
			name:          "error getting metadata",
			topic:         "test-topic",
			minPartitions: 3,
			setupMock: func(m *MockAdminClient) {
				m.On("GetMetadata", mock.AnythingOfType("*string"), false, 5000).Return(nil, fmt.Errorf("metadata fetch failed"))
			},
			wantErr:  true,
			errorMsg: "failed to get topic metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockAdminClient)
			tt.setupMock(mockClient)

			err := EnsureTopicPartitions(mockClient, tt.topic, tt.minPartitions)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
			mockClient.AssertExpectations(t)
		})
	}
}
