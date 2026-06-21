package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MonalFinbox/seatrush/internal/hold"
	"github.com/MonalFinbox/seatrush/internal/middleware"
	"github.com/MonalFinbox/seatrush/internal/respond"
	"github.com/MonalFinbox/seatrush/internal/store"
	"github.com/MonalFinbox/seatrush/internal/ws"
)

type createBookingRequest struct {
	HoldID      string         `json:"holdId"`
	PaymentMock map[string]any `json:"paymentMock"`
}

// CreateBooking converts a valid hold into a permanent booking. It verifies the
// hold exists and belongs to the caller, then persists the booking in a single
// DB transaction (seats -> booked, booking + join rows + payment). Only after
// the DB commits does it consume the Redis hold and broadcast seat.booked, so a
// failure can't leave the two stores disagreeing.
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	userID := h.userID(r)

	var req createBookingRequest
	if err := decode(r, &req); err != nil || req.HoldID == "" {
		respond.Error(w, http.StatusBadRequest, "holdId is required")
		return
	}

	held, err := h.Holds.Get(r.Context(), req.HoldID)
	if err != nil {
		if errors.Is(err, hold.ErrHoldNotFound) {
			respond.Error(w, http.StatusConflict, "hold expired or not found")
			return
		}
		respond.Error(w, http.StatusServiceUnavailable, "could not read hold, try again")
		return
	}
	if held.UserID != userID {
		respond.Error(w, http.StatusForbidden, "this hold belongs to someone else")
		return
	}

	booking, err := h.Store.CreateBooking(r.Context(), userID, held.EventID, held.SeatIDs, "mock_ticket_"+req.HoldID)
	if err != nil {
		if errors.Is(err, store.ErrSeatUnavailable) {
			respond.Error(w, http.StatusConflict, "one or more seats are no longer available")
			return
		}
		serverError(w)
		return
	}

	// DB is committed — now clear the hold and tell watchers the seats are sold.
	if _, err := h.Holds.Consume(r.Context(), req.HoldID); err != nil {
		// Non-fatal: the seats are booked in Postgres regardless. The hold's
		// TTL will clean up the Redis entries even if this call failed.
		_ = err
	}
	h.Hub.PublishMany(r.Context(), held.EventID, ws.EventBooked, held.SeatIDs)

	respond.JSON(w, http.StatusCreated, booking)
}

// ListBookings returns the caller's own bookings.
func (h *Handler) ListBookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.Store.ListBookingsByUser(r.Context(), h.userID(r))
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, bookings)
}

// GetBooking returns a booking. Visible to its owner or any admin.
func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	booking, err := h.Store.GetBooking(r.Context(), chi.URLParam(r, "bookingId"))
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	if booking.UserID != h.userID(r) && middleware.RoleFrom(r.Context()) != "admin" {
		respond.Error(w, http.StatusForbidden, "not your booking")
		return
	}
	respond.JSON(w, http.StatusOK, booking)
}

// CancelBooking cancels a booking (owner or admin), frees its seats, and
// broadcasts seat.released.
func (h *Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	bookingID := chi.URLParam(r, "bookingId")

	booking, err := h.Store.GetBooking(r.Context(), bookingID)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	if booking.UserID != h.userID(r) && middleware.RoleFrom(r.Context()) != "admin" {
		respond.Error(w, http.StatusForbidden, "not your booking")
		return
	}

	seatIDs, err := h.Store.CancelBooking(r.Context(), bookingID)
	if err != nil {
		notFoundOr500(w, err)
		return
	}

	h.Hub.PublishMany(r.Context(), booking.EventID, ws.EventReleased, seatIDs)
	respond.JSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
