// Package store is the persistence layer: every SQL query lives here, so
// handlers never touch the database directly.
package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a lookup matches no rows. Handlers translate
// this into a 404.
var ErrNotFound = errors.New("not found")

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}
