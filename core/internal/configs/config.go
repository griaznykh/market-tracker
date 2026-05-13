package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DSN              string
	TICKERS          []string
	GRPC_PORT        uint
	GRPC_REFLECTION  bool
	HTTP_PORT        uint
	INVEST_API_TOKEN string
	JWT_SECRET       string
	JWT_DURATION     time.Duration
}

func NewConfig() *Config {
	var grpc_port uint64 = 8000
	var http_port uint64 = 8001
	var grpc_reflection bool = false

	db_username := getEnv("DB_USERNAME", "")
	db_password := getEnv("DB_PASSWORD", "")
	db_host := getEnv("DB_HOST", "postgres")
	db_port := getEnv("DB_PORT", "5432")
	db_name := getEnv("DB_NAME", "marketdata")

	invest_api_token := getEnv("INVEST_API_TOKEN", "")

	if reflection := getEnv("REFLECTION", "false"); reflection != "" {
		grpc_reflection, _ = strconv.ParseBool(reflection)
	}

	if grpc_port_str := getEnv("GRPC_PORT", "8000"); grpc_port_str != "" {
		grpc_port, _ = strconv.ParseUint(grpc_port_str, 10, 64)
	}

	if http_port_str := getEnv("HTTP_PORT", "8001"); http_port_str != "" {
		http_port, _ = strconv.ParseUint(http_port_str, 10, 64)
	}

	jwt_secret := getEnv("JWT_SECRET", "")
	jwt_duration_str := getEnv("JWT_DURATION", "1d")
	jwt_duration, err := time.ParseDuration(jwt_duration_str)
	if err != nil {
		jwt_duration = 24 * time.Hour
	}

	return &Config{
		DSN:              fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", db_username, db_password, db_host, db_port, db_name),
		TICKERS:          strings.Split(os.Getenv("TICKERS"), ","),
		GRPC_REFLECTION:  grpc_reflection,
		GRPC_PORT:        uint(grpc_port),
		HTTP_PORT:        uint(http_port),
		INVEST_API_TOKEN: invest_api_token,
		JWT_DURATION:     jwt_duration,
		JWT_SECRET:       jwt_secret,
	}
}

func getEnv(key string, defaultValue string) string {
	if value, isExist := os.LookupEnv(key); isExist {
		return value
	}

	return defaultValue
}
