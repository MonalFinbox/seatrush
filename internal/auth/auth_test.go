package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	require.NoError(t, err)
	require.NotEqual(t, "correct horse battery staple", hash, "hash must not equal plaintext")

	require.True(t, CheckPassword(hash, "correct horse battery staple"))
	require.False(t, CheckPassword(hash, "wrong password"))
}

func TestAccessTokenRoundTrip(t *testing.T) {
	secret := "test-secret"
	tok, err := GenerateAccessToken(secret, "user-123", "attendee", time.Hour)
	require.NoError(t, err)

	claims, err := ParseAccessToken(secret, tok)
	require.NoError(t, err)
	require.Equal(t, "user-123", claims.UserID)
	require.Equal(t, "attendee", claims.Role)
}

func TestAccessTokenRejectsWrongSecret(t *testing.T) {
	tok, err := GenerateAccessToken("real-secret", "user-123", "attendee", time.Hour)
	require.NoError(t, err)

	_, err = ParseAccessToken("attacker-secret", tok)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestAccessTokenRejectsExpired(t *testing.T) {
	secret := "test-secret"
	tok, err := GenerateAccessToken(secret, "user-123", "attendee", -time.Minute) // already expired
	require.NoError(t, err)

	_, err = ParseAccessToken(secret, tok)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestRefreshTokenHashIsDeterministic(t *testing.T) {
	tok, err := GenerateRefreshToken()
	require.NoError(t, err)
	require.Equal(t, HashRefreshToken(tok), HashRefreshToken(tok), "same token must hash identically")
	require.NotEqual(t, tok, HashRefreshToken(tok), "stored hash must differ from raw token")
}
