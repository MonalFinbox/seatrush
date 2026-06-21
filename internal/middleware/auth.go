// Package middleware holds the app's custom HTTP middleware: authentication
// (verify the JWT) and authorization (check the role).
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/MonalFinbox/seatrush/internal/auth"
	"github.com/MonalFinbox/seatrush/internal/respond"
)

type ctxKey string

const (
	userIDKey ctxKey = "userID"
	roleKey   ctxKey = "role"
)

// Authenticator validates the Bearer token and stashes the user id + role in
// the request context. Anything behind it can trust those values.
func Authenticator(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				respond.Error(w, http.StatusUnauthorized, "missing or malformed Authorization header")
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			claims, err := auth.ParseAccessToken(secret, tokenStr)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, roleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole rejects any request whose role isn't in the allowed set. It must
// run after Authenticator.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := allowed[RoleFrom(r.Context())]; !ok {
				respond.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFrom returns the authenticated user id, or "" if unauthenticated.
func UserIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// RoleFrom returns the authenticated user's role, or "" if unauthenticated.
func RoleFrom(ctx context.Context) string {
	if v, ok := ctx.Value(roleKey).(string); ok {
		return v
	}
	return ""
}
