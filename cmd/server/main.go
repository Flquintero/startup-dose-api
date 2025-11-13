package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"startupdose.com/cmd/server/router"

)

func main() {
	// Create HTTP server
	mux := router.Setup()
	srv := &http.Server{
		Addr:           ":8080",
		Handler:        mux,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Channel for server errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Server starting on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			errChan <- err
		}
	}()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		// Graceful shutdown on signal
	case err := <-errChan:
		// Server error occurred
		fmt.Fprintf(os.Stderr, "Fatal server error: %v\n", err)
		os.Exit(1)
	}

	// Graceful shutdown with timeout
	fmt.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server forced to shutdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server stopped")
}
