package infra

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for LineBot Backend.
type Config struct {
	Addr                  string
	FirestoreProjectID    string
	FirestoreEmulatorHost string
	InternalGRPCAddr      string
	InternalAppID         string
	InternalBuilderID     int
	ServerReadTimeout     time.Duration
	ServerWriteTimeout    time.Duration
}

// LoadConfigFromEnv loads configuration from environment variables.
func LoadConfigFromEnv() Config {
	return Config{
		Addr:                  getEnvWithDefault("LINEBOT_ADDR", ":8083"),
		FirestoreProjectID:    getEnvWithDefault("LINEBOT_FIRESTORE_PROJECT_ID", "dailo-467502"),
		FirestoreEmulatorHost: os.Getenv("LINEBOT_FIRESTORE_EMULATOR_HOST"),
		InternalGRPCAddr:      getEnvWithDefault("LINEBOT_INTERNAL_GRPC_ADDR", "localhost:9091"),
		InternalAppID:         getEnvWithDefault("LINEBOT_INTERNAL_APP_ID", "linebot-app"),
		InternalBuilderID:     getEnvInt("LINEBOT_INTERNAL_BUILDER_ID", 4),
		ServerReadTimeout:     getEnvDuration("LINEBOT_SERVER_READ_TIMEOUT", 10*time.Second),
		ServerWriteTimeout:    getEnvDuration("LINEBOT_SERVER_WRITE_TIMEOUT", 5*time.Minute),
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
