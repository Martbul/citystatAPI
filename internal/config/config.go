	// internal/config/config.go
package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL         string
	ClerkSecretKey      string
	ClerkWebhookSecret  string
	UploadThingSecret   string
	Port                string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		ClerkSecretKey:     os.Getenv("CLERK_SECRET_KEY"),
		ClerkWebhookSecret: os.Getenv("CLERK_WEBHOOK_SECRET"),
		UploadThingSecret:  os.Getenv("UPLOADTHING_SECRET"),
		Port:               getEnvOrDefault("PORT", "3333"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.ClerkSecretKey == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}