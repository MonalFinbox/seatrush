// Package main — demo.go
//
// Seeds throwaway demo data so the app looks alive without manual setup:
//   - one demo organizer account (demo@seatrush.com / demo1234)
//   - 12 venues claimed by that organizer
//   - 12 published events (one per venue, dated in the future)
//   - 100-200 randomly-placed seats per event across four sections
//
// Every insert is idempotent: running `make seed` twice is safe.
// This file is intentionally separate from main.go — it is demo scaffolding,
// not production seeding logic.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Demo organizer — local dev only, never commit real credentials here.
const (
	demoEmail    = "demo@seatrush.com"
	demoPassword = "demo1234"
)

// demoEventSpec is the data for one event to be created.
type demoEventSpec struct {
	title       string
	description string
	daysFromNow int // event_date = today + N days
}

var demoEventSpecs = []demoEventSpec{
	{
		"Rock Legends Reunion",
		"The greatest rock bands of the decade return for one unforgettable night.",
		10,
	},
	{
		"Jazz Night Live",
		"An intimate evening of smooth jazz with the city's finest musicians.",
		14,
	},
	{
		"Stand-Up Comedy Gala",
		"A lineup of 8 comedians for a night of non-stop laughs.",
		18,
	},
	{
		"EDM Festival Night",
		"Bass-heavy beats and laser shows — the underground rave goes mainstream.",
		22,
	},
	{
		"Classical Symphony Evening",
		"A full orchestra performs Beethoven, Mozart, and Vivaldi.",
		28,
	},
	{
		"Hip-Hop Showcase",
		"Live performances from emerging and established hip-hop artists.",
		32,
	},
	{
		"Tech Startup Summit",
		"Keynotes, panels, and networking with founders and investors.",
		38,
	},
	{
		"Indie Film Screening",
		"Award-winning short films followed by a director Q&A session.",
		45,
	},
	{
		"Pop Icons Concert",
		"The biggest pop hits of the last 20 years, live on stage.",
		52,
	},
	{
		"Night of Opera",
		"A world-class soprano performs arias from the greatest operas ever written.",
		60,
	},
	{
		"Dance & Beats Night",
		"DJ sets, live dance performances, and an open dance floor until 2 AM.",
		67,
	},
	{
		"Alternative Rock Fest",
		"Three stages, 15 bands, one incredible afternoon of alt-rock.",
		75,
	},
}

// seedDemoData is the single entry point called from main().
func seedDemoData(db *pgxpool.Pool) {
	ctx := context.Background()
	organizerID := seedDemoOrganizer(ctx, db)
	venueIDs := claimDemoVenues(ctx, db, organizerID, len(demoEventSpecs))
	seedDemoEvents(ctx, db, organizerID, venueIDs)
	log.Println("Demo data seeded.")
}

// ─── organizer ───────────────────────────────────────────────────────────────

func seedDemoOrganizer(ctx context.Context, db *pgxpool.Pool) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt demo organizer: %v", err)
	}

	// ON CONFLICT DO NOTHING won't return the id, so we check with errors.Is
	// and fall back to a SELECT.
	var id string
	err = db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, 'organizer', 'active')
		ON CONFLICT (email) DO NOTHING
		RETURNING id
	`, demoEmail, string(hash)).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		// Row already existed; just get the id.
		err = db.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, demoEmail).Scan(&id)
	}
	if err != nil {
		log.Fatalf("seed demo organizer: %v", err)
	}

	log.Printf("Demo organizer: %s (id %s)", demoEmail, id)
	return id
}

// ─── venues ──────────────────────────────────────────────────────────────────

// claimDemoVenues picks n venues (unclaimed, or already owned by the demo
// organizer) and marks them as claimed. Returns their ids in stable order.
func claimDemoVenues(ctx context.Context, db *pgxpool.Pool, organizerID string, n int) []string {
	rows, err := db.Query(ctx, `
		SELECT id FROM venues
		WHERE claim_status = 'unclaimed' OR organizer_id = $1
		ORDER BY created_at
		LIMIT $2
	`, organizerID, n)
	if err != nil {
		log.Fatalf("list venues for claiming: %v", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("scan venue id: %v", err)
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		log.Fatalf("venue rows error: %v", rows.Err())
	}

	// Claim only the ones not yet claimed (UPDATE is a no-op if already claimed
	// by this organizer because claim_status would already be 'claimed').
	for _, id := range ids {
		if _, err := db.Exec(ctx, `
			UPDATE venues
			SET claim_status = 'claimed', organizer_id = $1
			WHERE id = $2 AND claim_status = 'unclaimed'
		`, organizerID, id); err != nil {
			log.Fatalf("claim venue %s: %v", id, err)
		}
	}

	log.Printf("Claimed %d venues for demo organizer", len(ids))
	return ids
}

// ─── events + seats ──────────────────────────────────────────────────────────

func seedDemoEvents(ctx context.Context, db *pgxpool.Pool, organizerID string, venueIDs []string) {
	now := time.Now()

	for i, spec := range demoEventSpecs {
		if i >= len(venueIDs) {
			break
		}
		venueID := venueIDs[i]
		eventDate := now.AddDate(0, 0, spec.daysFromNow)

		// Reuse an existing active event for this venue if one was already
		// created by a previous seed run.
		var eventID string
		err := db.QueryRow(ctx, `
			SELECT id FROM events
			WHERE venue_id = $1
			  AND status NOT IN ('cancelled', 'completed')
			LIMIT 1
		`, venueID).Scan(&eventID)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// No active event yet — create one.
			err = db.QueryRow(ctx, `
				INSERT INTO events
					(venue_id, organizer_id, title, description, event_date, status)
				VALUES ($1, $2, $3, $4, $5, 'published')
				RETURNING id
			`, venueID, organizerID, spec.title, spec.description, eventDate).Scan(&eventID)
			if err != nil {
				log.Fatalf("insert event %q: %v", spec.title, err)
			}
			log.Printf("  Event: %s", spec.title)

		case err != nil:
			log.Fatalf("check event for venue %s: %v", venueID, err)

		default:
			log.Printf("  Event already exists for venue, skipping: %s", spec.title)
		}

		seedDemoSeats(ctx, db, eventID, spec.title)
	}
}

// seedDemoSeats populates an event with 100-200 seats across four sections.
// Skips entirely if the event already has seats.
func seedDemoSeats(ctx context.Context, db *pgxpool.Pool, eventID, eventTitle string) {
	var existing int
	if err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM seats WHERE event_id = $1`, eventID,
	).Scan(&existing); err != nil {
		log.Fatalf("count seats for %s: %v", eventTitle, err)
	}
	if existing > 0 {
		log.Printf("    %d seats already exist, skipping %s", existing, eventTitle)
		return
	}

	sections := randomSeatLayout()
	total := 0
	for _, sec := range sections {
		for _, row := range sec.rows {
			for num := 1; num <= sec.perRow; num++ {
				if _, err := db.Exec(ctx, `
					INSERT INTO seats (event_id, section, seat_row, number, price)
					VALUES ($1, $2, $3, $4, $5)
					ON CONFLICT ON CONSTRAINT uniq_seat_in_event DO NOTHING
				`, eventID, sec.name, row, fmt.Sprintf("%d", num), sec.price); err != nil {
					log.Fatalf("insert seat [%s/%s/#%d]: %v", sec.name, row, num, err)
				}
				total++
			}
		}
	}
	log.Printf("    %d seats → %s", total, eventTitle)
}

// ─── seat layout helpers ──────────────────────────────────────────────────────

type seatSection struct {
	name   string
	rows   []string
	perRow int
	price  float64
}

// randomSeatLayout returns a four-section layout totalling 114–208 seats,
// which comfortably covers the 100-200 target for any random draw.
//
//	VIP     : 2-3 rows × 10-14 seats  =  20- 42
//	Floor A : 4-5 rows × 12-16 seats  =  48- 80
//	Floor B : 3-4 rows × 10-14 seats  =  30- 56
//	Balcony : 2-3 rows ×  8-10 seats  =  16- 30
//	────────────────────────────────────────────
//	Total                               114-208
func randomSeatLayout() []seatSection {
	return []seatSection{
		{
			name:   "VIP",
			rows:   makeRows(2 + rand.Intn(2)),       // 2 or 3
			perRow: 10 + rand.Intn(5),                // 10-14
			price:  float64(3500 + rand.Intn(1501)),  // 3500-5000
		},
		{
			name:   "Floor A",
			rows:   makeRows(4 + rand.Intn(2)),       // 4 or 5
			perRow: 12 + rand.Intn(5),                // 12-16
			price:  float64(1500 + rand.Intn(1001)),  // 1500-2500
		},
		{
			name:   "Floor B",
			rows:   makeRows(3 + rand.Intn(2)),       // 3 or 4
			perRow: 10 + rand.Intn(5),                // 10-14
			price:  float64(800 + rand.Intn(701)),    // 800-1500
		},
		{
			name:   "Balcony",
			rows:   makeRows(2 + rand.Intn(2)),       // 2 or 3
			perRow: 8 + rand.Intn(3),                 // 8-10
			price:  float64(300 + rand.Intn(501)),    // 300-800
		},
	}
}

// makeRows returns n alphabetic row labels: ["A", "B", "C", ...].
func makeRows(n int) []string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = string(rune('A' + i))
	}
	return rows
}
