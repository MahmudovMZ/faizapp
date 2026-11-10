package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

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
	Token        string        `json:"token"`
	BotMode      string        `json:"botMode"`
	AdminTGID    int64         `json:"adminTGID"`
	PhotoTimeout time.Duration `json:"photoTimeout"`
	PhotoWorkers int           `json:"photoWorkers"`
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
	tgAdminValue := os.Getenv("ADMIN_TELEGRAM_ID")
	photoTimeout := 60 * time.Second // если в .env ничего нет
	timeValue := os.Getenv("PHOTO_TIMEOUT")
	if tgToken == "" || botMode == "" || tgAdminValue == "" {
		return nil, fmt.Errorf("TGBOTAPI_TOKEN, BOT_MODE and ADMIN_TELEGRAM_ID are required")
	}
	tgAdmin, err := strconv.ParseInt(tgAdminValue, 10, 64)
	if err != nil {
		return nil, errors.New("ADMIN_TELEGRAM_ID must be a valid integer")
	}

	if tgAdmin <= 0 {
		return nil, errors.New("tgAdmin must be a positive integer")
	}

	if botMode != "polling" {
		return nil, fmt.Errorf("BOT_MODE must be 'polling'")
	}
	if timeValue != "" {
		dur, err := time.ParseDuration(timeValue)
		if err != nil {
			return nil, fmt.Errorf("PHOTO_TIMEOUT must be a duration like 60s: %w", err)
		}
		if dur <= 0 {
			return nil, errors.New("PHOTO_TIMEOUT must be greater than zero")
		}
		photoTimeout = dur
	}
	photoWorkers := 4
	workersValue := os.Getenv("PHOTO_WORKERS")
	if workersValue != "" {
		numWorkers, err := strconv.Atoi(workersValue)
		if err != nil {
			return nil, fmt.Errorf("PHOTO_WORKERS must be a valid integer: %w", err)
		}
		if numWorkers < 1 || numWorkers > 8 {
			return nil, errors.New("PHOTO_WORKERS must be between 1 and 8")
		}
		photoWorkers = numWorkers
	}
	return &BotConfig{
		Token:        tgToken,
		BotMode:      botMode,
		AdminTGID:    tgAdmin,
		PhotoTimeout: photoTimeout,
		PhotoWorkers: photoWorkers,
	}, nil
}
