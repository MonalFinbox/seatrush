package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MonalFinbox/seatrush/internal/respond"
)

// ListVenues returns seeded venues, optionally filtered by ?status=unclaimed.
func (h *Handler) ListVenues(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && status != "unclaimed" && status != "claimed" {
		respond.Error(w, http.StatusBadRequest, "status must be 'unclaimed' or 'claimed'")
		return
	}

	venues, err := h.Store.ListVenues(r.Context(), status)
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, venues)
}

// GetVenue returns one venue by id.
func (h *Handler) GetVenue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "venueId")
	venue, err := h.Store.GetVenue(r.Context(), id)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, venue)
}
