// Package ws provides per-event WebSocket broadcasting.
//
// Clients subscribe to one event. When a seat changes, the API publishes a
// message to a Redis channel (ws:event:{eventId}); every app instance is
// subscribed, receives it, and fans it out to the local sockets in that room.
// Routing broadcasts through Redis pub/sub means multiple API instances stay in
// sync without talking to each other directly.
package ws

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Event is the message shape pushed to clients.
type Event struct {
	Type      string    `json:"type"` // seat.held | seat.released | seat.booked
	SeatID    string    `json:"seatId"`
	Timestamp time.Time `json:"timestamp"`
}

const (
	EventHeld     = "seat.held"
	EventReleased = "seat.released"
	EventBooked   = "seat.booked"
)

type Hub struct {
	rdb   *redis.Client
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{} // eventID -> set of clients
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rdb:   rdb,
		rooms: make(map[string]map[*Client]struct{}),
	}
}

func channel(eventID string) string { return "ws:event:" + eventID }

// Run subscribes to all event channels and fans messages out to local clients.
// It blocks until ctx is cancelled, so run it in its own goroutine.
func (h *Hub) Run(ctx context.Context) {
	sub := h.rdb.PSubscribe(ctx, "ws:event:*")
	defer sub.Close()

	for {
		msg, err := sub.ReceiveMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			log.Printf("ws: subscription error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		eventID := strings.TrimPrefix(msg.Channel, "ws:event:")
		h.deliverLocal(eventID, []byte(msg.Payload))
	}
}

// Publish broadcasts a seat event to every subscriber of that event, across all
// instances, via Redis.
func (h *Hub) Publish(ctx context.Context, eventID, eventType, seatID string) {
	payload, err := json.Marshal(Event{
		Type:      eventType,
		SeatID:    seatID,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return
	}
	if err := h.rdb.Publish(ctx, channel(eventID), payload).Err(); err != nil {
		log.Printf("ws: publish failed: %v", err)
	}
}

// PublishMany is a convenience for broadcasting the same event type for several
// seats (e.g. a multi-seat hold or release).
func (h *Hub) PublishMany(ctx context.Context, eventID, eventType string, seatIDs []string) {
	for _, id := range seatIDs {
		h.Publish(ctx, eventID, eventType, id)
	}
}

func (h *Hub) deliverLocal(eventID string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[eventID] {
		select {
		case c.send <- payload:
		default:
			// Client's buffer is full — drop the message rather than block the
			// whole fan-out. A slow client shouldn't stall everyone else.
		}
	}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[c.eventID] == nil {
		h.rooms[c.eventID] = make(map[*Client]struct{})
	}
	h.rooms[c.eventID][c] = struct{}{}
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[c.eventID]; ok {
		delete(room, c)
		if len(room) == 0 {
			delete(h.rooms, c.eventID)
		}
	}
}
