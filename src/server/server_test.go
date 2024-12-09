package server

import (
	"io"
	"net/http"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	// Start the server in a goroutine
	port := 8081
	go Start(port)

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Send a request to the /health endpoint
	resp, err := http.Get("http://localhost:8081/health")
	if err != nil {
		t.Fatalf("Failed to reach server: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expected := "OK"
	if string(body) != expected {
		t.Errorf("Expected body %q, got %q", expected, string(body))
	}
}
