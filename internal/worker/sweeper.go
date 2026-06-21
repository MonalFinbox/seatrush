// Package worker holds background goroutines that run independently of any
// HTTP request.
package worker

import (
	"context"
	"log"
	"time"

	"github.com/MonalFinbox/seatrush/internal/hold"
	"github.com/MonalFinbox/seatrush/internal/ws"
)

// Sweeper periodically releases expired holds and broadcasts seat.released, so
// seats free themselves the instant their TTL passes without any client asking.
type Sweeper struct {
	holds    *hold.Manager
	hub      *ws.Hub
	interval time.Duration
}

func NewSweeper(holds *hold.Manager, hub *ws.Hub, interval time.Duration) *Sweeper {
	return &Sweeper{holds: holds, hub: hub, interval: interval}
}

// Run ticks until ctx is cancelled. On each tick it sweeps expired holds across
// every event and broadcasts what it freed. SweepExpired's ZREM makes the work
// idempotent, so a seat is never released or broadcast twice.
func (s *Sweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("sweeper: running every %s", s.interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("sweeper: stopping")
			return
		case <-ticker.C:
			released, err := s.holds.SweepExpired(ctx)
			if err != nil {
				log.Printf("sweeper: sweep failed: %v", err)
				continue
			}
			for _, rel := range released {
				s.hub.PublishMany(ctx, rel.EventID, ws.EventReleased, rel.SeatIDs)
			}
		}
	}
}
