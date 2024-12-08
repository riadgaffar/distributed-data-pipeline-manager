package utils

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// ExitFunc allows os.Exit to be overridden for testing.
var ExitFunc = os.Exit

// SetupSignalHandler sets up a signal handler for graceful shutdown.
func SetupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("INFO: Caught termination signal, shutting down...")
		ExitFunc(0) // Use the overridable ExitFunc
	}()
}
