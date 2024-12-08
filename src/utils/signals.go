package utils

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// setupSignalHandler captures termination signals for graceful shutdown
func SetupSignalHandler() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("INFO: Caught termination signal, shutting down...")
		os.Exit(0)
	}()

	return sigChan
}
