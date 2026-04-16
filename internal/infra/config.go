package infra

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for LineBot Backend.
type Config struct {
	Addr                       string
	FirestoreProjectID         string
	FirestoreEmulatorHost      string
	InternalGRPCAddr           string
	InternalAppID              string
	InternalBuilderID          int
	GoogleCalendarEnabled      bool
	GoogleCalendarID           string
	GoogleCalendarTimeZone     string
	GoogleOAuthCredentialsFile string
	GoogleOAuthTokenFile       string
	ServerReadTimeout          time.Duration
	ServerWriteTimeout         time.Duration
	LineChannelSecret          string
	LineChannelAccessToken     string
	LineBotUserID              string
}

// LoadConfigFromEnv loads configuration from environment variables.
func LoadConfigFromEnv() Config {
	return Config{
		Addr:                       getEnvWithDefault("LINEBOT_ADDR", ":8083"),
		FirestoreProjectID:         getEnvWithDefault("LINEBOT_FIRESTORE_PROJECT_ID", "dailo-467502"),
		FirestoreEmulatorHost:      os.Getenv("LINEBOT_FIRESTORE_EMULATOR_HOST"),
		InternalGRPCAddr:           getEnvWithDefault("LINEBOT_INTERNAL_GRPC_ADDR", "localhost:9091"),
		InternalAppID:              getEnvWithDefault("LINEBOT_INTERNAL_APP_ID", "linebot-app"),
		InternalBuilderID:          getEnvInt("LINEBOT_INTERNAL_BUILDER_ID", 4),
		GoogleCalendarEnabled:      getEnvBool("LINEBOT_GOOGLE_CALENDAR_ENABLED", false),
		GoogleCalendarID:           os.Getenv("LINEBOT_GOOGLE_CALENDAR_ID"),
		GoogleCalendarTimeZone:     getEnvWithDefault("LINEBOT_GOOGLE_CALENDAR_TIME_ZONE", "Asia/Taipei"),
		GoogleOAuthCredentialsFile: os.Getenv("LINEBOT_GOOGLE_OAUTH_CREDENTIALS_FILE"),
		GoogleOAuthTokenFile:       os.Getenv("LINEBOT_GOOGLE_OAUTH_TOKEN_FILE"),
		ServerReadTimeout:          getEnvDuration("LINEBOT_SERVER_READ_TIMEOUT", 10*time.Second),
		ServerWriteTimeout:         getEnvDuration("LINEBOT_SERVER_WRITE_TIMEOUT", 5*time.Minute),
		LineChannelSecret:          os.Getenv("LINEBOT_LINE_CHANNEL_SECRET"),
		LineChannelAccessToken:     os.Getenv("LINEBOT_LINE_CHANNEL_ACCESS_TOKEN"),
		LineBotUserID:              os.Getenv("LINEBOT_LINE_BOT_USER_ID"),
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
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
