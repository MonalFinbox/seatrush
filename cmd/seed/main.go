package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/MonalFinbox/seatrush/internal/config"
	database "github.com/MonalFinbox/seatrush/internal/db"
)

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

	// ON CONFLICT DO NOTHING makes this idempotent:
	// running the seed twice won't create a duplicate or error out.
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
	venues := []struct {
		name     string
		address  string
		city     string
		capacity int
	}{
		{"NSCI Dome", "Worli", "Mumbai", 12000},
		{"Bhavan's College Grounds", "Andheri West", "Mumbai", 8000},
		{"Jio World Garden", "BKC", "Mumbai", 20000},
		{"Whistling Woods Auditorium", "Goregaon East", "Mumbai", 1500},
		{"Nehru Centre", "Worli", "Mumbai", 1000},
	}

	for _, v := range venues {
		_, err := db.Exec(context.Background(), `
			INSERT INTO venues (name, address, city, capacity)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT DO NOTHING
		`, v.name, v.address, v.city, v.capacity)
		if err != nil {
			log.Fatalf("failed to seed venue %s: %v", v.name, err)
		}
	}

	log.Printf("Seeded %d venues", len(venues))
}
