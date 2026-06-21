package models

import "time"

// User is a person in the system. Password hash is never serialized to JSON.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Venue struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	Capacity    int       `json:"capacity"`
	ClaimStatus string    `json:"claimStatus"`
	OrganizerID *string   `json:"organizerId"` // pointer: null when unclaimed
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type VenueRegistrationRequest struct {
	ID              string     `json:"id"`
	VenueID         string     `json:"venueId"`
	OrganizerID     string     `json:"organizerId"`
	DocumentMock    string     `json:"documentMock"`
	Status          string     `json:"status"`
	ReviewedBy      *string    `json:"reviewedBy"`
	ReviewedAt      *time.Time `json:"reviewedAt"`
	RejectionReason *string    `json:"rejectionReason"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type Event struct {
	ID          string    `json:"id"`
	VenueID     string    `json:"venueId"`
	OrganizerID string    `json:"organizerId"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	EventDate   time.Time `json:"eventDate"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Seat carries a live Status that combines Postgres truth (available/booked)
// with Redis-derived "held". The DB column itself only ever holds
// available/booked; the handler overlays "held" when serving the seat map.
type Seat struct {
	ID        string    `json:"id"`
	EventID   string    `json:"eventId"`
	Section   string    `json:"section"`
	Row       string    `json:"row"`
	Number    string    `json:"number"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Booking struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	EventID     string    `json:"eventId"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"totalAmount"`
	SeatIDs     []string  `json:"seatIds,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Payment struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	BookingID *string   `json:"bookingId"`
	Type      string    `json:"type"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Reference string    `json:"reference"`
	CreatedAt time.Time `json:"createdAt"`
}
