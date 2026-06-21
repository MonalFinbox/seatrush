// Package handler contains the HTTP handlers for every endpoint. Handlers
// decode the request, call the store / hold manager / hub, and write JSON.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/MonalFinbox/seatrush/internal/auth"
	"github.com/MonalFinbox/seatrush/internal/cache"
	"github.com/MonalFinbox/seatrush/internal/config"
	"github.com/MonalFinbox/seatrush/internal/hold"
	"github.com/MonalFinbox/seatrush/internal/middleware"
	"github.com/MonalFinbox/seatrush/internal/models"
	"github.com/MonalFinbox/seatrush/internal/respond"
	"github.com/MonalFinbox/seatrush/internal/store"
	"github.com/MonalFinbox/seatrush/internal/ws"
)

type Handler struct {
	Store *store.Store
	Holds *hold.Manager
	Hub   *ws.Hub
	Cache *cache.Cache
	Cfg   *config.Config
}

func New(s *store.Store, h *hold.Manager, hub *ws.Hub, c *cache.Cache, cfg *config.Config) *Handler {
	return &Handler{Store: s, Holds: h, Hub: hub, Cache: c, Cfg: cfg}
}

// decode reads a JSON body into dst, rejecting unknown fields so typos in
// client payloads surface as errors instead of being silently ignored.
func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// userID is a shortcut for the authenticated user's id from the request.
func (h *Handler) userID(r *http.Request) string {
	return middleware.UserIDFrom(r.Context())
}

// currentUser loads the authenticated user from the DB using the id the auth
// middleware put in the context.
func (h *Handler) currentUser(ctx context.Context) (*models.User, error) {
	id := middleware.UserIDFrom(ctx)
	if id == "" {
		return nil, store.ErrNotFound
	}
	return h.Store.GetUserByID(ctx, id)
}

// tokenPair is the auth response returned by register/login/refresh/activate.
type tokenPair struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
}

// issueTokens mints a fresh access + refresh token pair and persists the
// refresh token's hash.
func (h *Handler) issueTokens(ctx context.Context, u *models.User) (*tokenPair, error) {
	access, err := auth.GenerateAccessToken(h.Cfg.JWTSecret, u.ID, u.Role, h.Cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(h.Cfg.RefreshTokenTTL)
	if err := h.Store.StoreRefreshToken(ctx, u.ID, auth.HashRefreshToken(refresh), expiresAt); err != nil {
		return nil, err
	}
	return &tokenPair{User: u, AccessToken: access, RefreshToken: refresh}, nil
}

// serverError logs nothing extra here but centralizes the 500 response.
func serverError(w http.ResponseWriter) {
	respond.Error(w, http.StatusInternalServerError, "internal server error")
}

// notFoundOr500 maps store.ErrNotFound to 404 and anything else to 500.
func notFoundOr500(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}
	serverError(w)
}
