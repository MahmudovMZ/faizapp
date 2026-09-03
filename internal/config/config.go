package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type DBConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
}

type BotConfig struct {
	Token   string `json:"token"`
	BotMode string `json:"botMode"`
}

type Config struct {
	Bot BotConfig
	DB  DBConfig
}

func Load() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil {
		return nil, err
	}
	tgBot, err := NewBotConfig()
	if err != nil {
		return nil, err
	}
	db, err := NewDBConfig()
	if err != nil {
		return nil, err
	}
	return &Config{
		Bot: *tgBot,
		DB:  *db,
	}, nil
}

func NewDBConfig() (*DBConfig, error) {
	host := os.Getenv("POSTGRES_HOST")
	portValue := os.Getenv("POSTGRES_PORT")
	name := os.Getenv("POSTGRES_DB")
	password := os.Getenv("POSTGRES_PASSWORD")
	user := os.Getenv("POSTGRES_USER")

	required := []struct {
		name  string
		value string
	}{
		{"POSTGRES_HOST", host},
		{"POSTGRES_PORT", portValue},
		{"POSTGRES_DB", name},
		{"POSTGRES_PASSWORD", password},
		{"POSTGRES_USER", user},
	}

	for _, item := range required {
		if item.value == "" {
			return nil, fmt.Errorf("%s is required", item.name)
		}
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return nil, fmt.Errorf("POSTGRES_PORT must be a valid integer: %w", err)
	}

	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("POSTGRES_PORT must be between 1 and 65535")
	}

	return &DBConfig{
		Username: user,
		Password: password,
		DBName:   name,
		Host:     host,
		Port:     port,
	}, nil
}

func NewBotConfig() (*BotConfig, error) {
	tgToken := os.Getenv("TGBOTAPI_TOKEN")
	botMode := os.Getenv("BOT_MODE")

	if tgToken == "" || botMode == "" {
		return &BotConfig{}, fmt.Errorf("TGBOTAPI_TOKEN and BOT_MODE are required")
	}

	if botMode != "polling" {
		return &BotConfig{}, fmt.Errorf("BOT_MODE must be 'polling'")
	}
	return &BotConfig{
		Token:   tgToken,
		BotMode: botMode,
	}, nil
}
