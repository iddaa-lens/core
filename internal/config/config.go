package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	External ExternalAPIConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ExternalAPIConfig struct {
	BaseURL string
	APIKey  string
	Timeout int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "localhost"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5433"),
			User:     getEnv("DB_USER", "iddaa"),
			Password: getEnv("DB_PASSWORD", "iddaa123"),
			DBName:   getEnv("DB_NAME", "iddaa_core"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		External: ExternalAPIConfig{
			BaseURL: getEnv("EXTERNAL_API_URL", ""),
			APIKey:  getEnv("EXTERNAL_API_KEY", ""),
			Timeout: getEnvAsInt("EXTERNAL_API_TIMEOUT", 30),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func (c *Config) DatabaseURL() string {
	// If DATABASE_URL is set, use it directly
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		return databaseURL
	}

	// Otherwise, construct from individual components
	return "postgres://" + c.Database.User + ":" + c.Database.Password +
		"@" + c.Database.Host + ":" + c.Database.Port +
		"/" + c.Database.DBName + "?sslmode=" + c.Database.SSLMode
}
