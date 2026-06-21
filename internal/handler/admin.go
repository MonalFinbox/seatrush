package handler

import (
	"net/http"

	"github.com/MonalFinbox/seatrush/internal/respond"
)

// AdminListEvents returns every event across all organizers.
func (h *Handler) AdminListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.Store.ListAllEvents(r.Context())
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, events)
}

// AdminListBookings returns every booking in the system.
func (h *Handler) AdminListBookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.Store.ListAllBookings(r.Context())
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, bookings)
}

// AdminListUsers returns every user.
func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Store.ListUsers(r.Context())
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, users)
}
