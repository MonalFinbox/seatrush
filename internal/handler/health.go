package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Health returns an http.HandlerFunc — a function that handles one specific request.
// It takes its dependencies (db, redis) as arguments and closes over them.
// This pattern is called a "closure-based handler" — common in Go when a handler
// needs access to shared resources without a global variable.
func Health(db *pgxpool.Pool, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbErr := db.Ping(context.Background())
		redisErr := redisClient.Ping(context.Background()).Err()

		status := map[string]string{
			"postgres": statusString(dbErr),
			"redis":    statusString(redisErr),
		}

		httpStatus := http.StatusOK
		if dbErr != nil || redisErr != nil {
			httpStatus = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		json.NewEncoder(w).Encode(status)
	}
}

func statusString(err error) string {
	if err != nil {
		return "Down"
	}
	return "Up & Running"
}
