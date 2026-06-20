package db

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
)

// Pool is a connection pool, not a single connection.
// pgx manages multiple connections internally you never open one manually.
func New(databaseURL string) (*pgxpool.Pool, error) {
    pool, err := pgxpool.New(context.Background(), databaseURL)
    if err != nil {
        return nil, fmt.Errorf("could not create db pool: %w", err)
    }

    // Ping verifies the pool can actually reach Postgres.
    if err := pool.Ping(context.Background()); err != nil {
        return nil, fmt.Errorf("could not ping db: %w", err)
    }

    return pool, nil
}