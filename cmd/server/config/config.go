package config

import (
	"log"
	"os"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port string

	// API Security
	APIKey string

	// OpenAI
	OpenAIAPIKey string

	// Supabase
	SupabaseURL         string
	SupabaseKey         string
	SupabaseServiceRole string

	// Logging
	LogLevel string

	// HTTP Client
	HTTPTimeout              string
	HTTPMaxIdleConns         string
	HTTPMaxIdleConnsPerHost  string
	HTTPMaxConnsPerHost      string

	// AWS S3
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	S3BucketName       string

	// ScreenshotOne API
	ScreenshotOneAPIKey string
}

// Load reads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		// Server
		Port: getEnv("PORT", "8080"),

		// API Security
		APIKey: getEnv("API_KEY", ""),

		// OpenAI
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),

		// Supabase
		SupabaseURL:         getEnv("SUPABASE_URL", ""),
		SupabaseKey:         getEnv("SUPABASE_KEY", ""),
		SupabaseServiceRole: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),

		// HTTP Client
		HTTPTimeout:             getEnv("HTTP_TIMEOUT", "5s"),
		HTTPMaxIdleConns:        getEnv("HTTP_MAX_IDLE_CONNS", "100"),
		HTTPMaxIdleConnsPerHost: getEnv("HTTP_MAX_IDLE_CONNS_PER_HOST", "10"),
		HTTPMaxConnsPerHost:     getEnv("HTTP_MAX_CONNS_PER_HOST", "100"),

		// AWS S3
		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3BucketName:       getEnv("S3_BUCKET_NAME", ""),

		// ScreenshotOne API
		ScreenshotOneAPIKey: getEnv("SCREENSHOTONE_API_KEY", ""),
	}

	// Validate required Supabase credentials
	if cfg.SupabaseURL == "" {
		log.Println("Warning: SUPABASE_URL not set")
	}
	if cfg.SupabaseKey == "" {
		log.Println("Warning: SUPABASE_KEY not set")
	}

	return cfg
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
