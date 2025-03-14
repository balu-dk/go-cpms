package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	ServerPort int
	APIPort    int
	OCPPPath   string

	// Database configuration
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// OCPP configuration
	HeartbeatInterval int

	// Logging
	LogLevel string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Server configuration
	serverPort, err := strconv.Atoi(getEnv("SERVER_PORT", "8887"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %v", err)
	}

	apiPort, err := strconv.Atoi(getEnv("API_PORT", "8888"))
	if err != nil {
		return nil, fmt.Errorf("invalid API_PORT: %v", err)
	}

	// Database configuration
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %v", err)
	}

	// OCPP configuration
	heartbeatInterval, err := strconv.Atoi(getEnv("HEARTBEAT_INTERVAL", "600"))
	if err != nil {
		return nil, fmt.Errorf("invalid HEARTBEAT_INTERVAL: %v", err)
	}

	return &Config{
		// Server configuration
		ServerPort: serverPort,
		APIPort:    apiPort,
		OCPPPath:   getEnv("OCPP_PATH", "/ocpp"),

		// Database configuration
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     dbPort,
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "cpms"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// OCPP configuration
		HeartbeatInterval: heartbeatInterval,

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}, nil
}

// GetDSN returns the PostgreSQL connection string
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// SetupLogger configures the global logger
func (c *Config) SetupLogger() {
	level, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

// Helper function to get environment variables with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
