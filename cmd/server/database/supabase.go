package database

import (
	"fmt"
	"log"
	"sync"

	"github.com/supabase-community/supabase-go"
	"startupdose.com/cmd/server/config"
)

var (
	client *supabase.Client
	once   sync.Once
)

// InitSupabase initializes the Supabase client
func InitSupabase(cfg *config.Config) error {
	var err error

	once.Do(func() {
		if cfg.SupabaseURL == "" {
			err = fmt.Errorf("supabase URL not provided")
			log.Println("Supabase client not initialized: missing URL")
			return
		}

		// Prefer service role key (bypasses RLS), fall back to anon key
		apiKey := cfg.SupabaseServiceRole
		keyType := "service role"
		if apiKey == "" {
			apiKey = cfg.SupabaseKey
			keyType = "anon"
		}

		if apiKey == "" {
			err = fmt.Errorf("supabase API key not provided")
			log.Println("Supabase client not initialized: missing API key")
			return
		}

		client, err = supabase.NewClient(cfg.SupabaseURL, apiKey, nil)
		if err != nil {
			log.Printf("Failed to initialize Supabase client: %v\n", err)
			return
		}

		log.Printf("Supabase client initialized successfully using %s key\n", keyType)
	})

	return err
}

// GetClient returns the Supabase client instance
func GetClient() *supabase.Client {
	if client == nil {
		log.Println("Warning: Supabase client is not initialized")
	}
	return client
}

// Close closes the Supabase client connection (if needed for cleanup)
func Close() error {
	// The supabase-go client doesn't require explicit closing
	// but we keep this function for consistency and future use
	log.Println("Supabase client cleanup completed")
	return nil
}
