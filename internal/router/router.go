package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/MonalFinbox/seatrush/internal/config"
	"github.com/MonalFinbox/seatrush/internal/handler"
	appmw "github.com/MonalFinbox/seatrush/internal/middleware"
)

// New builds the full HTTP router: middleware, the public surface, the
// authenticated /api/v1 surface gated by role, and the WebSocket endpoint.
func New(h *handler.Handler, db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Operational endpoint, outside the versioned API.
	r.Get("/health", handler.Health(db, rdb))

	auth := appmw.Authenticator(cfg.JWTSecret)

	r.Route("/api/v1", func(r chi.Router) {
		// ---- Public ----
		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/admin/login", h.AdminLogin)
		r.Post("/auth/refresh", h.Refresh)

		r.Get("/venues", h.ListVenues)
		r.Get("/venues/{venueId}", h.GetVenue)

		r.Get("/events", h.ListEvents)
		r.Get("/events/{eventId}", h.GetEvent)
		r.Get("/events/{eventId}/seats", h.GetSeats)

		// ---- Authenticated ----
		r.Group(func(r chi.Router) {
			r.Use(auth)

			r.Get("/users/me", h.Me)

			// Organizer onboarding
			r.With(appmw.RequireRole("organizer")).
				Post("/auth/organizer/activate", h.ActivateOrganizer)

			// Venue registration (organizer)
			r.With(appmw.RequireRole("organizer")).
				Post("/venues/{venueId}/registration-requests", h.SubmitVenueRequest)
			r.With(appmw.RequireRole("organizer")).
				Get("/venues/registration-requests/me", h.MyVenueRequests)

			// Events (organizer creates; organizer/admin manage)
			r.With(appmw.RequireRole("organizer")).Post("/events", h.CreateEvent)
			r.With(appmw.RequireRole("organizer", "admin")).Patch("/events/{eventId}", h.PatchEvent)
			r.With(appmw.RequireRole("organizer", "admin")).Post("/events/{eventId}/publish", h.PublishEvent)
			r.With(appmw.RequireRole("organizer", "admin")).Post("/events/{eventId}/cancel", h.CancelEvent)
			r.With(appmw.RequireRole("organizer", "admin")).Post("/events/{eventId}/seats", h.CreateSeats)

			// Holds (attendee)
			r.With(appmw.RequireRole("attendee")).Post("/events/{eventId}/holds", h.CreateHold)
			r.With(appmw.RequireRole("attendee")).Delete("/holds/{holdId}", h.ReleaseHold)

			// Bookings
			r.With(appmw.RequireRole("attendee")).Post("/bookings", h.CreateBooking)
			r.With(appmw.RequireRole("attendee")).Get("/bookings", h.ListBookings)
			// owner-or-admin checks happen inside these handlers
			r.Get("/bookings/{bookingId}", h.GetBooking)
			r.Post("/bookings/{bookingId}/cancel", h.CancelBooking)

			// Admin dashboard
			r.With(appmw.RequireRole("admin")).Get("/admin/venue-registration-requests", h.AdminListVenueRequests)
			r.With(appmw.RequireRole("admin")).Post("/admin/venue-registration-requests/{requestId}/approve", h.ApproveVenueRequest)
			r.With(appmw.RequireRole("admin")).Post("/admin/venue-registration-requests/{requestId}/reject", h.RejectVenueRequest)
			r.With(appmw.RequireRole("admin")).Get("/admin/events", h.AdminListEvents)
			r.With(appmw.RequireRole("admin")).Get("/admin/bookings", h.AdminListBookings)
			r.With(appmw.RequireRole("admin")).Get("/admin/users", h.AdminListUsers)
		})
	})

	// Realtime, under its own versioned prefix.
	r.Get("/ws/v1/events/{eventId}", h.EventWebSocket)

	return r
}
