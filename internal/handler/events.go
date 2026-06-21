package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/MonalFinbox/seatrush/internal/models"
	"github.com/MonalFinbox/seatrush/internal/respond"
)

// Cache-aside settings for event reads.
const (
	eventDetailTTL = 60 * time.Second
	eventListTTL   = 30 * time.Second
)

// invalidateEventCaches busts the detail key plus every list variant after a
// write, so the next read repopulates from the database.
func (h *Handler) invalidateEventCaches(ctx context.Context, eventID string) {
	keys := []string{"event:" + eventID}
	for _, st := range []string{"", "draft", "published", "cancelled", "completed"} {
		keys = append(keys, "events:list:"+st)
	}
	h.Cache.Del(ctx, keys...)
}

// canManageEvent allows the owning organizer or any admin.
func canManageEvent(user *models.User, event *models.Event) bool {
	return user.Role == "admin" || event.OrganizerID == user.ID
}

type createEventRequest struct {
	VenueID     string    `json:"venueId"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	EventDate   time.Time `json:"eventDate"`
}

// CreateEvent creates an event at a venue the organizer owns. The DB's partial
// unique index rejects a second active event on the same venue.
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	user, err := h.currentUser(r.Context())
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	if user.Status != "active" {
		respond.Error(w, http.StatusForbidden, "activate your account first")
		return
	}

	var req createEventRequest
	if err := decode(r, &req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.VenueID == "" || req.Title == "" || req.EventDate.IsZero() {
		respond.Error(w, http.StatusBadRequest, "venueId, title and eventDate are required")
		return
	}

	venue, err := h.Store.GetVenue(r.Context(), req.VenueID)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	// Ownership: the organizer must own the venue they're creating an event at.
	if venue.OrganizerID == nil || *venue.OrganizerID != user.ID {
		respond.Error(w, http.StatusForbidden, "you do not own this venue")
		return
	}

	event, err := h.Store.CreateEvent(r.Context(), req.VenueID, user.ID, req.Title, req.Description, req.EventDate)
	if err != nil {
		if isUniqueViolation(err) {
			respond.Error(w, http.StatusConflict, "this venue already has an active event")
			return
		}
		serverError(w)
		return
	}
	h.invalidateEventCaches(r.Context(), event.ID)
	respond.JSON(w, http.StatusCreated, event)
}

// ListEvents returns events, optionally filtered by ?status=published. Public.
// Cache-aside: served from Redis on a hit, from Postgres (then cached) on a miss.
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	key := "events:list:" + status

	var cached []models.Event
	if found, _ := h.Cache.GetJSON(r.Context(), key, &cached); found {
		log.Printf("cache HIT  %s", key)
		respond.JSON(w, http.StatusOK, cached)
		return
	}
	log.Printf("cache MISS %s", key)

	events, err := h.Store.ListEvents(r.Context(), status)
	if err != nil {
		serverError(w)
		return
	}
	h.Cache.SetJSON(r.Context(), key, events, eventListTTL)
	respond.JSON(w, http.StatusOK, events)
}

// GetEvent returns one event. Public. Cache-aside on key event:{id}.
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "eventId")
	key := "event:" + id

	var cached models.Event
	if found, _ := h.Cache.GetJSON(r.Context(), key, &cached); found {
		log.Printf("cache HIT  %s", key)
		respond.JSON(w, http.StatusOK, cached)
		return
	}
	log.Printf("cache MISS %s", key)

	event, err := h.Store.GetEvent(r.Context(), id)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	h.Cache.SetJSON(r.Context(), key, event, eventDetailTTL)
	respond.JSON(w, http.StatusOK, event)
}

type patchEventRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	EventDate   *time.Time `json:"eventDate"`
}

// PatchEvent updates an event. Owning organizer or admin only.
func (h *Handler) PatchEvent(w http.ResponseWriter, r *http.Request) {
	user, event, ok := h.loadManageableEvent(w, r)
	if !ok {
		return
	}
	_ = user

	var req patchEventRequest
	if err := decode(r, &req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.Store.UpdateEvent(r.Context(), event.ID, req.Title, req.Description, req.EventDate)
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	h.invalidateEventCaches(r.Context(), event.ID)
	respond.JSON(w, http.StatusOK, updated)
}

// PublishEvent makes an event bookable. Owning organizer or admin only.
func (h *Handler) PublishEvent(w http.ResponseWriter, r *http.Request) {
	_, event, ok := h.loadManageableEvent(w, r)
	if !ok {
		return
	}
	updated, err := h.Store.SetEventStatus(r.Context(), event.ID, "published")
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	h.invalidateEventCaches(r.Context(), event.ID)
	respond.JSON(w, http.StatusOK, updated)
}

// CancelEvent cancels an event, which frees its venue for a new event because
// cancelled rows fall out of the one-active-event partial index.
func (h *Handler) CancelEvent(w http.ResponseWriter, r *http.Request) {
	_, event, ok := h.loadManageableEvent(w, r)
	if !ok {
		return
	}
	updated, err := h.Store.SetEventStatus(r.Context(), event.ID, "cancelled")
	if err != nil {
		notFoundOr500(w, err)
		return
	}
	h.invalidateEventCaches(r.Context(), event.ID)
	respond.JSON(w, http.StatusOK, updated)
}

// loadManageableEvent loads the event named in the URL and verifies the caller
// may manage it. It writes the error response itself and returns ok=false when
// the caller should stop.
func (h *Handler) loadManageableEvent(w http.ResponseWriter, r *http.Request) (*models.User, *models.Event, bool) {
	user, err := h.currentUser(r.Context())
	if err != nil {
		notFoundOr500(w, err)
		return nil, nil, false
	}
	event, err := h.Store.GetEvent(r.Context(), chi.URLParam(r, "eventId"))
	if err != nil {
		notFoundOr500(w, err)
		return nil, nil, false
	}
	if !canManageEvent(user, event) {
		respond.Error(w, http.StatusForbidden, "you do not manage this event")
		return nil, nil, false
	}
	return user, event, true
}
