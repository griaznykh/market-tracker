package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	POSTGRES_DSN string
	TICKERS      []string
	PORT         string
}

func NewConfig() *Config {
	db_username := getEnv("DB_USERNAME", "")
	db_password := getEnv("DB_PASSWORD", "")
	db_host := getEnv("DB_HOST", "postgres")
	db_port := getEnv("DB_PORT", "5432")
	db_name := getEnv("DB_NAME", "marketdata")

	return &Config{
		POSTGRES_DSN: fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", db_username, db_password, db_host, db_port, db_name),
		TICKERS:      strings.Split(os.Getenv("TICKERS"), ","),
		PORT:         os.Getenv("PORT"),
	}
}

func getEnv(key string, defaultValue string) string {
	if value, isExist := os.LookupEnv(key); isExist {
		return value
	}

	return defaultValue
}
