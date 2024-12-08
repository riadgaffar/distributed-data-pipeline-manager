package utils

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	exitFunc   = os.Exit                 // Allows overriding in tests
	signalChan = make(chan os.Signal, 1) // Global signal channel
)

// SetupSignalHandler handles OS signals and triggers a shutdown.
func SetupSignalHandler() {
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Println("INFO: Caught termination signal, shutting down...")
		exitFunc(0)
	}()
}
