package main

import (
	"log"
	"net/http"

	"github.com/MonalFinbox/seatrush/internal/cache"
	"github.com/MonalFinbox/seatrush/internal/config"
	database "github.com/MonalFinbox/seatrush/internal/db"
	"github.com/MonalFinbox/seatrush/internal/router"
)

func main() {
	secrets := config.Load()

	// DB POOL
	dbPool, err := database.New(secrets.DatabaseURL)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	defer dbPool.Close()

	// Redis Client
	redisClient, err := cache.New(secrets.RedisAddr)
	if err != nil {
		log.Fatalf("redis init failed: %v", err)
	}
	defer redisClient.Close()

    // Router
	r := router.New(dbPool, redisClient)

	log.Printf("SeatRush listening on :%s", secrets.ServerPort)
	if err := http.ListenAndServe(":"+secrets.ServerPort, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
