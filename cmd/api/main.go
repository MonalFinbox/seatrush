package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/MonalFinbox/seatrush/internal/cache"
	"github.com/MonalFinbox/seatrush/internal/config"
	database "github.com/MonalFinbox/seatrush/internal/db"
	"github.com/MonalFinbox/seatrush/internal/handler"
	"github.com/MonalFinbox/seatrush/internal/hold"
	"github.com/MonalFinbox/seatrush/internal/router"
	"github.com/MonalFinbox/seatrush/internal/store"
	"github.com/MonalFinbox/seatrush/internal/worker"
	"github.com/MonalFinbox/seatrush/internal/ws"
)

func main() {
	secrets := config.Load()

	// DB pool
	dbPool, err := database.New(secrets.DatabaseURL)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	defer dbPool.Close()

	// Redis client
	redisClient, err := cache.New(secrets.RedisAddr)
	if err != nil {
		log.Fatalf("redis init failed: %v", err)
	}
	defer redisClient.Close()

	// Wire the layers together.
	st := store.New(dbPool)
	holds := hold.New(redisClient, secrets.HoldTTL)
	hub := ws.NewHub(redisClient)
	cacheStore := cache.NewCache(redisClient)
	h := handler.New(st, holds, hub, cacheStore, secrets)

	// rootCtx is cancelled on SIGINT/SIGTERM and stops the background workers.
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Background goroutines: the pub/sub fan-out hub and the hold sweeper.
	go hub.Run(rootCtx)
	go worker.NewSweeper(holds, hub, 10*time.Second).Run(rootCtx)

	srv := &http.Server{
		Addr:    ":" + secrets.ServerPort,
		Handler: router.New(h, dbPool, redisClient, secrets),
	}

	// Run the HTTP server in its own goroutine so main can wait for a signal.
	go func() {
		log.Printf("SeatRush listening on :%s", secrets.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-rootCtx.Done() // block until a shutdown signal arrives
	log.Println("shutting down...")

	// Give in-flight requests a few seconds to finish before forcing exit.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("bye")
}
