package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for the app.
type Config struct {
	DatabaseURL string
	RedisAddr   string
	JWTSecret   string
	ServerPort  string
}

// Load reads .env and returns a populated Config.
func Load() *Config {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	// Also read real environment variables — these win over .env values.
	// Useful when deploying: set vars in the shell, skip the file entirely.
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("could not load .env file: %v", err)
	}

	return &Config{
		DatabaseURL: viper.GetString("DATABASE_URL"),
		RedisAddr:   viper.GetString("REDIS_ADDR"),
		JWTSecret:   viper.GetString("JWT_SECRET"),
		ServerPort:  viper.GetString("SERVER_PORT"),
	}
}
