package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for the app.
type Config struct {
	DatabaseURL   string
	RedisAddr     string
	JWTSecret     string
	ServerPort    string
	AdminEmail    string
	AdminPassword string
	AdminKey      string // required alongside password to log in as admin

	// Token lifetimes
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Business knobs
	PlatformFee float64       // mock organizer registration fee
	HoldTTL     time.Duration // how long a seat hold survives before auto-release

	// CORS allowed origins for the browser frontend.
	CORSOrigins []string
}

func Load() *Config {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Defaults so the app boots even if these aren't in .env.
	viper.SetDefault("ACCESS_TOKEN_TTL_MIN", 15)
	viper.SetDefault("REFRESH_TOKEN_TTL_HOURS", 168) // 7 days
	viper.SetDefault("PLATFORM_FEE", 4999.00)
	viper.SetDefault("HOLD_TTL_SECONDS", 300) // 5 minutes
	viper.SetDefault("CORS_ORIGINS", "http://localhost:5173,http://localhost:4173")

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

		AccessTokenTTL:  time.Duration(viper.GetInt("ACCESS_TOKEN_TTL_MIN")) * time.Minute,
		RefreshTokenTTL: time.Duration(viper.GetInt("REFRESH_TOKEN_TTL_HOURS")) * time.Hour,
		PlatformFee:     viper.GetFloat64("PLATFORM_FEE"),
		HoldTTL:         time.Duration(viper.GetInt("HOLD_TTL_SECONDS")) * time.Second,
		CORSOrigins:     splitAndTrim(viper.GetString("CORS_ORIGINS")),
	}
}

// splitAndTrim turns "a, b ,c" into ["a","b","c"].
func splitAndTrim(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
