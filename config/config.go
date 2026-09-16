package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server    ServerConfig
	Log       LogConfig
	Database  DatabaseConfig
	Anthropic AnthropicConfig
	Twilio    TwilioConfig
	OpenAI    OpenAIConfig
}

type AnthropicConfig struct {
	APIKey string
	Model  string
}

type TwilioConfig struct {
	AccountSID       string
	AuthToken        string
	ApiKeyId         string
	ApiKeySecret     string
	WorkspaceSid     string
	WebhookBaseUrl   string
	WebsocketBaseUrl string
}

type ServerConfig struct {
	Host string
	Port int
}

type LogConfig struct {
	Level string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type OpenAIConfig struct {
	ApiKey            string
	WebhookSigningKey string
}

func Load() Config {
	return Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "zennify"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Anthropic: AnthropicConfig{
			APIKey: getEnv("ANTHROPIC_API_KEY", ""),
		},
		Twilio: TwilioConfig{
			AccountSID:       getEnv("TWILIO_ACCOUNT_SID", ""),
			AuthToken:        getEnv("TWILIO_AUTH_TOKEN", ""),
			ApiKeyId:         getEnv("TWILIO_API_KEY_ID", ""),
			ApiKeySecret:     getEnv("TWILIO_API_KEY_SECRET", ""),
			WorkspaceSid:     getEnv("TWILIO_WORKSPACE_SID", ""),
			WebhookBaseUrl:   getEnv("TWILIO_WEBHOOK_BASE_URL", ""),
			WebsocketBaseUrl: getEnv("TWILIO_WEBSOCKET_BASE_URL", ""),
		},
		OpenAI: OpenAIConfig{
			ApiKey:            getEnv("OPEN_AI_API_KEY", ""),
			WebhookSigningKey: getEnv("OPEN_AI_WEBHOOK_SIGNING_KEY", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
