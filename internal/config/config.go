package config

import (
	"log"

	"github.com/spf13/viper"
)

// all the runtime needed
type Config struct {
	DatabaseURL   string
	RedisAddr     string
	JWTSecret     string
	ServerPort    string
	AdminEmail    string
	AdminPassword string
	AdminKey      string // required alongside password to log in as admin
}

func Load() *Config {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("could not load .env file: %v", err)
	}

	return &Config{
		DatabaseURL:   viper.GetString("DATABASE_URL"),
		RedisAddr:     viper.GetString("REDIS_ADDR"),
		JWTSecret:     viper.GetString("JWT_SECRET"),
		ServerPort:    viper.GetString("SERVER_PORT"),
		AdminEmail:    viper.GetString("ADMIN_EMAIL"),
		AdminPassword: viper.GetString("ADMIN_PASSWORD"),
		AdminKey:      viper.GetString("ADMIN_KEY"),
	}
}
