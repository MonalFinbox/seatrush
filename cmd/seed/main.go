package main

import (
	"context"
	"encoding/json"
	"log"

	// embed is a blank import — it registers the //go:embed directive.
	// Without this import the directive is silently ignored.
	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/MonalFinbox/seatrush/internal/config"
	database "github.com/MonalFinbox/seatrush/internal/db"
)

// Go reads this file at compile time and stores its bytes in venuesJSON.
// No file path needed at runtime — the data is baked into the binary.
//
//go:embed data/venues.json
var venuesJSON []byte

type venue struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	City     string `json:"city"`
	Capacity int    `json:"capacity"`
}

func main() {
	secrets := config.Load()

	db, err := database.New(secrets.DatabaseURL)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	seedAdmin(db, secrets)
	seedVenues(db)

	log.Println("Seed complete.")
}

func seedAdmin(db *pgxpool.Pool, secrets *config.Config) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secrets.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}

	_, err = db.Exec(context.Background(), `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, 'admin', 'active')
		ON CONFLICT (email) DO NOTHING
	`, secrets.AdminEmail, string(hash))
	if err != nil {
		log.Fatalf("failed to seed admin: %v", err)
	}

	log.Printf("Admin seeded: %s", secrets.AdminEmail)
}

func seedVenues(db *pgxpool.Pool) {
	var venues []venue
	if err := json.Unmarshal(venuesJSON, &venues); err != nil {
		log.Fatalf("failed to parse venues.json: %v", err)
	}

	for _, v := range venues {
		_, err := db.Exec(context.Background(), `
			INSERT INTO venues (name, address, city, capacity)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT DO NOTHING
		`, v.Name, v.Address, v.City, v.Capacity)
		if err != nil {
			log.Fatalf("failed to seed venue %s: %v", v.Name, err)
		}
	}

	log.Printf("Seeded %d venues", len(venues))
}
