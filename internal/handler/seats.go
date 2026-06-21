package handler

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MonalFinbox/seatrush/internal/respond"
	"github.com/MonalFinbox/seatrush/internal/store"
)

type seatInput struct {
	Section string  `json:"section"`
	Row     string  `json:"row"`
	Number  string  `json:"number"`
	Price   float64 `json:"price"`
}

type bulkSeatsRequest struct {
	Seats []seatInput `json:"seats"`
}

// CreateSeats bulk-defines an event's seat map. Allowed once: a second attempt
// is rejected so the map can't be silently duplicated.
func (h *Handler) CreateSeats(w http.ResponseWriter, r *http.Request) {
	_, event, ok := h.loadManageableEvent(w, r)
	if !ok {
		return
	}

	var req bulkSeatsRequest
	if err := decode(r, &req); err != nil || len(req.Seats) == 0 {
		respond.Error(w, http.StatusBadRequest, "at least one seat is required")
		return
	}

	count, err := h.Store.CountSeats(r.Context(), event.ID)
	if err != nil {
		serverError(w)
		return
	}
	if count > 0 {
		respond.Error(w, http.StatusConflict, "seat map already defined for this event")
		return
	}

	seats := make([]store.SeatInput, 0, len(req.Seats))
	for _, s := range req.Seats {
		if s.Section == "" || s.Row == "" || s.Number == "" || s.Price < 0 {
			respond.Error(w, http.StatusBadRequest, "each seat needs section, row, number and a non-negative price")
			return
		}
		seats = append(seats, store.SeatInput{Section: s.Section, Row: s.Row, Number: s.Number, Price: s.Price})
	}

	if err := h.Store.BulkCreateSeats(r.Context(), event.ID, seats); err != nil {
		if isUniqueViolation(err) {
			respond.Error(w, http.StatusBadRequest, "duplicate seats in payload (same section/row/number)")
			return
		}
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusCreated, map[string]int{"created": len(seats)})
}

// GetSeats returns the full seat map with live status. Postgres supplies
// available/booked; Redis supplies which available seats are currently held,
// which we overlay as "held" in the response. Public.
func (h *Handler) GetSeats(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")

	seats, err := h.Store.ListSeats(r.Context(), eventID)
	if err != nil {
		serverError(w)
		return
	}

	// Overlay held state from Redis. If Redis is unreachable we still return
	// the DB truth (available/booked) rather than failing the whole read.
	heldIDs, err := h.Holds.HeldSeats(r.Context(), eventID)
	if err != nil {
		log.Printf("seats: could not read held seats from redis: %v", err)
	}
	held := make(map[string]struct{}, len(heldIDs))
	for _, id := range heldIDs {
		held[id] = struct{}{}
	}

	for i := range seats {
		if seats[i].Status == "available" {
			if _, isHeld := held[seats[i].ID]; isHeld {
				seats[i].Status = "held"
			}
		}
	}
	respond.JSON(w, http.StatusOK, seats)
}
