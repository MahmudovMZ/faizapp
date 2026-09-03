package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type DBConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
	Host     string `json:"Host"`
	Port     string `json:"Port"`
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
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")
	password := os.Getenv("DB_PASS")
	user := os.Getenv("DB_USER")

	if host == "" || port == "" || name == "" || password == "" || user == "" {
		log.Fatal("[CONFIGURING] some of the DB environment variables was not set")
		return &DBConfig{}, nil
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

//func ReadConfig(path string) error {
//	b, err := os.ReadFile(path)
//	if err != nil {
//		return err
//	}
//	return json.Unmarshal(b, &conf)
//}
//
//func GetConf() *Config {
//	return &conf
//}
