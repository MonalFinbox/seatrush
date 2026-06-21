package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MonalFinbox/seatrush/internal/hold"
	"github.com/MonalFinbox/seatrush/internal/respond"
	"github.com/MonalFinbox/seatrush/internal/ws"
)

type createHoldRequest struct {
	SeatIDs []string `json:"seatIds"`
}

type createHoldResponse struct {
	HoldID    string   `json:"holdId"`
	SeatIDs   []string `json:"seatIds"`
	ExpiresAt string   `json:"expiresAt"`
}

// CreateHold atomically holds one or more seats for the attendee. The Lua
// script in the hold manager guarantees two concurrent requests for the same
// seat can never both succeed. On success it broadcasts seat.held to everyone
// watching the event.
func (h *Handler) CreateHold(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	userID := h.userID(r)

	var req createHoldRequest
	if err := decode(r, &req); err != nil || len(req.SeatIDs) == 0 {
		respond.Error(w, http.StatusBadRequest, "seatIds is required")
		return
	}

	event, err := h.Store.GetEvent(r.Context(), eventID)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	if event.Status != "published" {
		respond.Error(w, http.StatusConflict, "event is not open for booking")
		return
	}

	// Verify every requested seat belongs to this event and is available in
	// Postgres (i.e. not already booked) before we touch Redis.
	seats, err := h.Store.ListSeats(r.Context(), eventID)
	if err != nil {
		serverError(w)
		return
	}
	available := make(map[string]struct{}, len(seats))
	for _, s := range seats {
		if s.Status == "available" {
			available[s.ID] = struct{}{}
		}
	}
	for _, id := range req.SeatIDs {
		if _, ok := available[id]; !ok {
			respond.Error(w, http.StatusConflict, "one or more seats are not available")
			return
		}
	}

	held, err := h.Holds.Create(r.Context(), eventID, userID, req.SeatIDs)
	if err != nil {
		if errors.Is(err, hold.ErrSeatTaken) {
			respond.Error(w, http.StatusConflict, "one or more seats are already held")
			return
		}
		// Redis unreachable or scripting error: fail closed. We never allow a
		// hold we can't guarantee is exclusive.
		respond.Error(w, http.StatusServiceUnavailable, "could not place hold, try again")
		return
	}

	h.Hub.PublishMany(r.Context(), eventID, ws.EventHeld, held.SeatIDs)

	respond.JSON(w, http.StatusCreated, createHoldResponse{
		HoldID:    held.ID,
		SeatIDs:   held.SeatIDs,
		ExpiresAt: held.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ReleaseHold manually frees a hold the attendee owns and broadcasts
// seat.released.
func (h *Handler) ReleaseHold(w http.ResponseWriter, r *http.Request) {
	holdID := chi.URLParam(r, "holdId")
	userID := h.userID(r)

	released, err := h.Holds.Release(r.Context(), holdID, userID)
	if err != nil {
		switch {
		case errors.Is(err, hold.ErrHoldNotFound):
			respond.Error(w, http.StatusNotFound, "hold not found or already expired")
		case errors.Is(err, hold.ErrNotOwner):
			respond.Error(w, http.StatusForbidden, "this hold belongs to someone else")
		default:
			respond.Error(w, http.StatusServiceUnavailable, "could not release hold, try again")
		}
		return
	}

	h.Hub.PublishMany(r.Context(), released.EventID, ws.EventReleased, released.SeatIDs)
	respond.JSON(w, http.StatusOK, map[string]string{"status": "released"})
}
