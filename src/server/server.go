package server

import (
	"log"
	"net/http"
	"strconv"
)

// Start initializes and starts the HTTP server.
func Start(port int) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := ":" + strconv.Itoa(port)
	log.Printf("INFO: HTTP server running on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("ERROR: Failed to start HTTP server: %v", err)
	}
}
