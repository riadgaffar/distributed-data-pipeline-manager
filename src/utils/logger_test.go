// logger_test.go
package utils

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel logrus.Level
	}{
		{
			name:          "debug level",
			level:         "debug",
			expectedLevel: logrus.DebugLevel,
		},
		{
			name:          "info level",
			level:         "info",
			expectedLevel: logrus.InfoLevel,
		},
		{
			name:          "warn level",
			level:         "warn",
			expectedLevel: logrus.WarnLevel,
		},
		{
			name:          "error level",
			level:         "error",
			expectedLevel: logrus.ErrorLevel,
		},
		{
			name:          "invalid level defaults to info",
			level:         "invalid",
			expectedLevel: logrus.InfoLevel,
		},
		{
			name:          "case insensitive level",
			level:         "DEBUG",
			expectedLevel: logrus.DebugLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset logger before each test
			logrus.SetLevel(logrus.InfoLevel)

			// Call function under test
			SetLogLevel(tt.level)

			// Verify level was set correctly
			assert.Equal(t, tt.expectedLevel, logrus.GetLevel())
		})
	}
}
