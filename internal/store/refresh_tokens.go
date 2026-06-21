package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// StoreRefreshToken persists the hash of an issued refresh token.
func (s *Store) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

// GetValidRefreshTokenUser looks up the owner of a refresh token, but only if
// it is neither expired nor revoked. Returns ErrNotFound otherwise.
func (s *Store) GetValidRefreshTokenUser(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := s.db.QueryRow(ctx, `
		SELECT user_id FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`,
		tokenHash,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
}

// RevokeRefreshToken marks a token revoked. Used on rotation and logout, this
// is what stateful refresh tokens buy us over plain stateless JWTs.
func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	)
	return err
}
