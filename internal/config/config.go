package config

import (
	"net"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	ServerPort       string
	ServerHost       string
	AppURL           string
	TelegramBotToken string
	SessionSecret    string
	DevLogin         bool
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DatabaseURL      string
	DataDir          string
	AdminTelegramIDs string
	WorkerEnabled    bool
	AdsEnabled       bool
	AdsGramBlockID   string
	AdsGramDebug     bool
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func Load() *Config {
	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		ServerHost:       getEnv("SERVER_HOST", "127.0.0.1"),
		AppURL:           getEnv("APP_URL", "http://localhost:"+getEnv("SERVER_PORT", "8080")),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		SessionSecret:    os.Getenv("SESSION_SECRET"),
		DevLogin:         os.Getenv("DEV_LOGIN") == "true",
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5433"),
		DBUser:           getEnv("DB_USER", "contgen_app"),
		DBPassword:       getEnv("DB_PASSWORD", "app_secret"),
		DBName:           getEnv("DB_NAME", "contgen_db"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DataDir:          getEnv("DATA_DIR", "data"),
		AdminTelegramIDs: os.Getenv("ADMIN_TELEGRAM_IDS"),
		WorkerEnabled:    getEnv("WORKER_ENABLED", "true") == "true",
		AdsEnabled:       os.Getenv("ADS_ENABLED") == "true",
		AdsGramBlockID:   strings.TrimSpace(os.Getenv("ADSGRAM_BLOCK_ID")),
		AdsGramDebug:     os.Getenv("ADSGRAM_DEBUG") == "true",
	}
}

func (c *Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword(c.DBUser, c.DBPassword), Host: net.JoinHostPort(c.DBHost, c.DBPort), Path: "/" + c.DBName}
	u.RawQuery = "sslmode=disable"
	return u.String()
}
