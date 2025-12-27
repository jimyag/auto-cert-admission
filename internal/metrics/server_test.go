package metrics

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	t.Run("with default path", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port: 8080,
		})

		if server.config.Port != 8080 {
			t.Errorf("Port: got %d, want %d", server.config.Port, 8080)
		}

		if server.config.Path != "/metrics" {
			t.Errorf("Path: got %q, want %q", server.config.Path, "/metrics")
		}
	})

	t.Run("with custom path", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port: 9090,
			Path: "/custom-metrics",
		})

		if server.config.Path != "/custom-metrics" {
			t.Errorf("Path: got %q, want %q", server.config.Path, "/custom-metrics")
		}
	})
}

func TestServer_Start(t *testing.T) {
	t.Run("starts and stops gracefully", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port: 19090, // Use high port to avoid conflicts
			Path: "/metrics",
		})

		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- server.Start(ctx)
		}()

		// Wait for server to start
		time.Sleep(100 * time.Millisecond)

		// Verify server is running
		resp, err := http.Get("http://localhost:19090/metrics")
		if err != nil {
			t.Fatalf("Failed to connect to metrics server: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// Stop the server
		cancel()

		// Wait for shutdown
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("Server returned error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Server did not shutdown in time")
		}
	})
}
