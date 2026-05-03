package main

import (
	"log"
	config "market-service/internal/configs"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}

	config := config.NewConfig()

	for _, ticker := range config.TICKERS {
		log.Print(ticker)
	}
}
