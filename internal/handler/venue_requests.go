package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MonalFinbox/seatrush/internal/respond"
)

type createRequestBody struct {
	DocumentMock string `json:"documentMock"`
}

// SubmitVenueRequest lets an active organizer claim an unclaimed venue. The
// partial unique index in the DB guarantees only one pending request per venue,
// so a concurrent second submission is rejected at the database level.
func (h *Handler) SubmitVenueRequest(w http.ResponseWriter, r *http.Request) {
	user, err := h.currentUser(r.Context())
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	if user.Status != "active" {
		respond.Error(w, http.StatusForbidden, "activate your account before claiming venues")
		return
	}

	venueID := chi.URLParam(r, "venueId")
	venue, err := h.Store.GetVenue(r.Context(), venueID)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	if venue.ClaimStatus != "unclaimed" {
		respond.Error(w, http.StatusConflict, "venue is already claimed")
		return
	}

	var body createRequestBody
	if err := decode(r, &body); err != nil || body.DocumentMock == "" {
		respond.Error(w, http.StatusBadRequest, "documentMock is required")
		return
	}

	req, err := h.Store.CreateVenueRequest(r.Context(), venueID, user.ID, body.DocumentMock)
	if err != nil {
		if isUniqueViolation(err) {
			respond.Error(w, http.StatusConflict, "a pending request already exists for this venue")
			return
		}
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusCreated, req)
}

// MyVenueRequests lists the calling organizer's own requests.
func (h *Handler) MyVenueRequests(w http.ResponseWriter, r *http.Request) {
	userID := h.userID(r)
	requests, err := h.Store.ListRequestsByOrganizer(r.Context(), userID)
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, requests)
}

// AdminListVenueRequests lists requests for review, optionally ?status=pending.
func (h *Handler) AdminListVenueRequests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	requests, err := h.Store.ListRequests(r.Context(), status)
	if err != nil {
		serverError(w)
		return
	}
	respond.JSON(w, http.StatusOK, requests)
}

// ApproveVenueRequest approves a pending request: the venue becomes claimed by
// the organizer, atomically, inside the store transaction.
func (h *Handler) ApproveVenueRequest(w http.ResponseWriter, r *http.Request) {
	adminID := h.userID(r)
	requestID := chi.URLParam(r, "requestId")

	req, err := h.Store.ApproveRequest(r.Context(), requestID, adminID)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, req)
}

type rejectBody struct {
	Reason string `json:"reason"`
}

// RejectVenueRequest rejects a pending request with a reason.
func (h *Handler) RejectVenueRequest(w http.ResponseWriter, r *http.Request) {
	adminID := h.userID(r)
	requestID := chi.URLParam(r, "requestId")

	var body rejectBody
	if err := decode(r, &body); err != nil || body.Reason == "" {
		respond.Error(w, http.StatusBadRequest, "reason is required")
		return
	}

	req, err := h.Store.RejectRequest(r.Context(), requestID, adminID, body.Reason)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, req)
}
