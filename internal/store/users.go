package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/MonalFinbox/seatrush/internal/models"
)

const userColumns = `id, email, password_hash, role, status, created_at, updated_at`

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, role, status string) (*models.User, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, $3, $4)
		RETURNING `+userColumns,
		email, passwordHash, role, status,
	)
	return scanUser(row)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	row := s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	row := s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (s *Store) SetUserStatus(ctx context.Context, id, status string) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}
