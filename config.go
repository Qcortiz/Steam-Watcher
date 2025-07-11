package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
}

// LoadConfig загружает переменные из .env
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("❌ Не удалось загрузить .env файл")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("❌ Переменная TELEGRAM_TOKEN не найдена")
	}

	return &Config{
		TelegramToken: token,
	}
}
