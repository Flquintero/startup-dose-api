package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"startupdose.com/cmd/server/config"
	"startupdose.com/cmd/server/database"
	"startupdose.com/cmd/server/router"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize Supabase client
	if err := database.InitSupabase(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Supabase: %v\n", err)
		// Note: We continue even if Supabase fails to initialize
		// This allows the server to run without Supabase if needed
	}

	// Ensure cleanup on exit
	defer database.Close()

	// Create HTTP server
	mux := router.Setup(cfg)
	srv := &http.Server{
		Addr:           ":" + cfg.Port,
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
