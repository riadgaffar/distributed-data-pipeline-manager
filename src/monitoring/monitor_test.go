package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMonitorAndScale(t *testing.T) {
	tests := []struct {
		name                 string
		cfg                  MonitorConfig
		mockGetConsumerLag   func(brokers, group, topic string) (int64, error)
		mockEnsurePartitions func(brokers, topic string, newPartitionCount int) error
		wantScalingCall      bool
		wantError            bool
	}{
		{
			name: "lag exceeds threshold, scaling triggered",
			cfg: MonitorConfig{
				Brokers:       "localhost:9092",
				Topic:         "test-topic",
				Group:         "test-group",
				Threshold:     1000,
				ScaleBy:       2,
				CheckInterval: 1 * time.Second,
			},
			mockGetConsumerLag: func(brokers, group, topic string) (int64, error) {
				return 1200, nil // Lag exceeds threshold
			},
			mockEnsurePartitions: func(brokers, topic string, newPartitionCount int) error {
				assert.Equal(t, "test-topic", topic)
				assert.Equal(t, 2, newPartitionCount)
				return nil
			},
			wantScalingCall: true,
		},
		{
			name: "lag below threshold, no scaling",
			cfg: MonitorConfig{
				Brokers:       "localhost:9092",
				Topic:         "test-topic",
				Group:         "test-group",
				Threshold:     1000,
				ScaleBy:       2,
				CheckInterval: 1 * time.Second,
			},
			mockGetConsumerLag: func(brokers, group, topic string) (int64, error) {
				return 500, nil // Lag below threshold
			},
			mockEnsurePartitions: func(brokers, topic string, newPartitionCount int) error {
				t.FailNow() // Should not be called
				return nil
			},
			wantScalingCall: false,
		},
		{
			name: "error fetching consumer lag",
			cfg: MonitorConfig{
				Brokers:       "localhost:9092",
				Topic:         "test-topic",
				Group:         "test-group",
				Threshold:     1000,
				ScaleBy:       2,
				CheckInterval: 1 * time.Second,
			},
			mockGetConsumerLag: func(brokers, group, topic string) (int64, error) {
				return 0, assert.AnError // Simulate lag fetch error
			},
			mockEnsurePartitions: func(brokers, topic string, newPartitionCount int) error {
				t.FailNow() // Should not be called
				return nil
			},
			wantScalingCall: false,
			wantError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scalingCalled bool

			// Wrap the mock function to track calls
			mockEnsurePartitions := func(brokers, topic string, newPartitionCount int) error {
				scalingCalled = true
				return tt.mockEnsurePartitions(brokers, topic, newPartitionCount)
			}

			// Run MonitorAndScale in a separate goroutine
			go func() {
				MonitorAndScale(tt.cfg, tt.mockGetConsumerLag, mockEnsurePartitions)
			}()

			// Wait briefly to allow MonitorAndScale to execute
			time.Sleep(2 * time.Second)

			assert.Equal(t, tt.wantScalingCall, scalingCalled)
		})
	}
}
