package config

import (
	"os"
)

type Config struct {
	Environment   string
	ServerAddress string
	DatabasePath  string
	SessionSecret string
	UploadPath    string
}

func Load() *Config {
	return &Config{
		Environment:   getEnv("APP_ENV", "development"),
		ServerAddress: getEnv("SERVER_ADDRESS", ":8080"),
		DatabasePath:  getEnv("DATABASE_PATH", "./data/dc_management.db"),
		SessionSecret: getEnv("SESSION_SECRET", "dev-secret-change-in-production"),
		UploadPath:    getEnv("UPLOAD_PATH", "./static/uploads"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
