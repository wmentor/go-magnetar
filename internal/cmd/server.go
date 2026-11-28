package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func serverFlow(state *plugin.State) error {
	// Read server configuration
	host := state.Config.String("server.host")
	port := state.Config.Int("server.port")
	cert := state.Config.String("server.cert")
	key := state.Config.String("server.key")
	tlsEnabled := state.Config.Bool("server.tls")

	// Build the address
	listenAddr := fmt.Sprintf("%s:%d", host, port)

	// Create HTTP handler
	mux := http.NewServeMux()

	// Add health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	setupUnixTime := time.Now().Unix()

	// Add models endpoint
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		modelID := state.Config.ProfileParamString("llm.model")
		maxModelLen := state.Config.ProfileParamInt("llm.context")

		response := map[string]any{
			"object": "list",
			"data": []map[string]any{
				{
					"id":            modelID,
					"object":        "model",
					"created":       setupUnixTime,
					"owned_by":      "go-magnetar",
					"model_type":    "llm",
					"max_model_len": maxModelLen,
				},
			},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			state.Printer.Error("Failed to encode models response: %v", err)
		}
	})

	// Create server
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Minute,
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	var serverStarted atomic.Bool

	go func() {
		var err error
		if tlsEnabled {
			state.Printer.Info("Starting HTTPS server on %s", listenAddr)
			err = server.ListenAndServeTLS(cert, key)
		} else {
			state.Printer.Info("Starting HTTP server on %s", listenAddr)
			err = server.ListenAndServe()
		}
		// Mark server as started only when it actually started
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			serverStarted.Store(true)
			state.Printer.Info("Server started successfully")
		}
		errChan <- err
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal or server error
	select {
	case sig := <-sigChan:
		state.Printer.Info("Received signal %v, shutting down server...", sig)
	case err := <-errChan:
		// Server error (not from graceful shutdown)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			state.Printer.Error("Server error: %v", err)
			return fmt.Errorf("server error: %w", err)
		}
		// Server started successfully but returned (e.g., ListenAndServe returned nil)
		if err == nil {
			state.Printer.Info("Server started successfully")
		}
		// Only mark as started when server actually started
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			serverStarted.Store(true)
		}
	}

	// Only perform graceful shutdown if server was started
	if serverStarted.Load() {
		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		state.Printer.Info("Shutting down server...")
		if err := server.Shutdown(ctx); err != nil {
			state.Printer.Error("Server shutdown error: %v", err)
			return fmt.Errorf("server shutdown error: %w", err)
		}

		state.Printer.Info("Server stopped")
	}

	return nil
}
