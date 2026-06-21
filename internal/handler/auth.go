package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MonalFinbox/seatrush/internal/auth"
	"github.com/MonalFinbox/seatrush/internal/respond"
)

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Register creates an attendee (active) or organizer (pending_payment). There
// is deliberately no code path here that can create an admin — admins are only
// ever seeded.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decode(r, &req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || len(req.Password) < 8 {
		respond.Error(w, http.StatusBadRequest, "email required and password must be at least 8 characters")
		return
	}
	if req.Role != "attendee" && req.Role != "organizer" {
		respond.Error(w, http.StatusBadRequest, "role must be 'attendee' or 'organizer'")
		return
	}

	status := "active"
	if req.Role == "organizer" {
		status = "pending_payment"
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		serverError(w)
		return
	}

	user, err := h.Store.CreateUser(r.Context(), req.Email, hash, req.Role, status)
	if err != nil {
		if isUniqueViolation(err) {
			respond.Error(w, http.StatusConflict, "email already registered")
			return
		}
		serverError(w)
		return
	}

	tokens, err := h.issueTokens(r.Context(), user)
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusCreated, tokens)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates an attendee or organizer. Admins are rejected here — they
// have their own secret route. Pending organizers are blocked until they pay.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decode(r, &req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		// Same response whether the user is missing or the password is wrong,
		// so we don't leak which emails are registered.
		respond.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		respond.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if user.Role == "admin" {
		respond.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if user.Role == "organizer" && user.Status == "pending_payment" {
		respond.Error(w, http.StatusForbidden, "account pending: pay the registration fee to activate")
		return
	}

	tokens, err := h.issueTokens(r.Context(), user)
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, tokens)
}

type adminLoginRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	AdminAccessKey string `json:"adminAccessKey"`
}

// AdminLogin is the secret admin route. Three things guard it: the static
// adminAccessKey, the requirement that the account already exists with role
// admin (only ever true for seeded accounts), and the password. A hidden URL
// alone is not the security — these checks are.
func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req adminLoginRequest
	if err := decode(r, &req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check the static key first; a wrong key never even hits the database.
	if req.AdminAccessKey == "" || req.AdminAccessKey != h.Cfg.AdminKey {
		respond.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	user, err := h.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || user.Role != "admin" || !auth.CheckPassword(user.PasswordHash, req.Password) {
		respond.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	tokens, err := h.issueTokens(r.Context(), user)
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, tokens)
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// Refresh rotates tokens: it validates the presented refresh token, revokes it,
// and issues a brand-new pair. Rotation means a stolen refresh token is only
// usable until the legitimate client next refreshes.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decode(r, &req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hashed := auth.HashRefreshToken(req.RefreshToken)
	userID, err := h.Store.GetValidRefreshTokenUser(r.Context(), hashed)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Revoke the old token before minting a new one (rotation).
	if err := h.Store.RevokeRefreshToken(r.Context(), hashed); err != nil {
		serverError(w)
		return
	}

	user, err := h.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		serverError(w)
		return
	}

	tokens, err := h.issueTokens(r.Context(), user)
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, tokens)
}

type activateRequest struct {
	PaymentMock map[string]any `json:"paymentMock"`
}

// ActivateOrganizer pays the mock platform fee and flips a pending organizer to
// active. It records a real payment row and re-issues tokens so the new access
// token reflects the active status immediately.
func (h *Handler) ActivateOrganizer(w http.ResponseWriter, r *http.Request) {
	user, err := h.currentUser(r.Context())
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	if user.Role != "organizer" {
		respond.Error(w, http.StatusForbidden, "only organizers can activate")
		return
	}
	if user.Status == "active" {
		respond.Error(w, http.StatusConflict, "account already active")
		return
	}

	var req activateRequest
	if err := decode(r, &req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ref := "mock_fee_" + user.ID
	if err := h.Store.CreatePlatformFeePayment(r.Context(), user.ID, h.Cfg.PlatformFee, ref); err != nil {
		serverError(w)
		return
	}
	if err := h.Store.SetUserStatus(r.Context(), user.ID, "active"); err != nil {
		serverError(w)
		return
	}
	user.Status = "active"

	tokens, err := h.issueTokens(r.Context(), user)
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, tokens)
}

// Me returns the current user's profile.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.currentUser(r.Context())
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, user)
}
