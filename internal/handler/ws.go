package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MonalFinbox/seatrush/internal/ws"
)

// EventWebSocket upgrades the connection and subscribes the client to live seat
// events for the given event. The client then receives seat.held /
// seat.released / seat.booked messages until it disconnects.
func (h *Handler) EventWebSocket(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	ws.Serve(h.Hub, eventID, w, r)
}
