package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/MonalFinbox/seatrush/internal/handler"
)

// builds the router with all middleware and routes registered.
func New(db *pgxpool.Pool, redisClient *redis.Client) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handler.Health(db, redisClient))

	return r
}
