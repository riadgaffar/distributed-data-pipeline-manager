package utils

import "github.com/sirupsen/logrus"

func SetLogLevel(level string) {
	logrus.Printf("DEBUG: Received log level: '%s'", level)

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Printf("WARN: Invalid log level '%s', defaulting to INFO\n", level)
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
}
