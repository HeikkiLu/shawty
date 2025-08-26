package config

import (
	"fmt"
	"strings"

	"github.com/sbowman/dotenv"
)

type Config struct {
	DBUser  string
	DBPass  string
	DBName  string
	DBHost  string
	DBPort  string
	SSLMode string
	BaseURL string
	Domain  string
	Port    string
}

func Load() (Config, error) {
	dotenv.Load()

	cfg := Config{
		DBUser:  dotenv.GetString("DB_USER"),
		DBPass:  dotenv.GetString("DB_USER_PASSWORD"),
		DBName:  dotenv.GetString("DB_NAME"),
		DBHost:  dotenv.GetString("DB_HOST"),
		DBPort:  dotenv.GetString("DB_PORT"),
		SSLMode: dotenv.GetString("DB_SSLMODE"),
		BaseURL: dotenv.GetString("BASE_URL"),
		Domain:  dotenv.GetString("DOMAIN"),
		Port:    dotenv.GetString("PORT"),
	}
	if !strings.HasSuffix(cfg.BaseURL, "/") {
		cfg.BaseURL += "/"
	}
	return cfg, nil
}

func (cfg Config) BindAddr() string {
	return fmt.Sprintf("%s:%s", cfg.Domain, cfg.Port)
}

func (cfg Config) DSN() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBHost, cfg.DBPort, cfg.SSLMode)
}
