# SeatRush

A live event ticket-booking backend where organizers are vetted and venues
claimed through an admin approval workflow, and attendees reserve seats with
real-time, concurrency-safe holds.

Built in **Go + PostgreSQL + Redis**. Single service, no orchestration — the
focus is depth on backend fundamentals: REST design, layered auth/RBAC,
relational modelling, caching, WebSockets, and (above all) getting concurrency
right.

> **The one rule that matters most:** two people can never hold or book the same
> seat at the same time, even under concurrent requests. Everything else in the
> system exists to support that guarantee.

---

## Table of contents

- [Architecture](#architecture)
- [Tech stack](#tech-stack)
- [Project layout](#project-layout)
- [Getting started](#getting-started)
- [Roles & the core flow](#roles--the-core-flow)
- [API reference](#api-reference)
- [How the hard parts work](#how-the-hard-parts-work)
- [Data model](#data-model)
- [Testing](#testing)

---

## Architecture

```
                         ┌──────────────────────────────┐
   HTTP / WebSocket  ──▶  │            chi router         │
                         │   middleware: JWT auth + RBAC  │
                         └───────────────┬───────────────┘
                                         │
                              ┌──────────▼──────────┐
                              │      handlers        │  decode → authorize → act → respond
                              └─────┬─────────┬──────┘
                  ┌─────────────────┘         └──────────────────┐
        ┌─────────▼─────────┐               ┌────────────────────▼───────────┐
        │   store (pgx)      │               │  hold.Manager / cache / ws.Hub  │
        │  all SQL lives here│               │           (Redis)               │
        └─────────┬─────────┘               └────────────────────┬───────────┘
                  │                                               │
            ┌─────▼─────┐                                   ┌─────▼─────┐
            │ PostgreSQL │                                   │   Redis   │
            │  (truth)   │                                   │ holds/pub │
            └───────────┘                                   └───────────┘

   background: ws.Hub (Redis pub/sub fan-out) + worker.Sweeper (expired holds)
```

**Separation of concerns** is deliberate:

- **handlers** decode requests, enforce authorization, and shape JSON — no SQL.
- **store** owns every SQL query and all transactions — nothing else touches the DB.
- **hold / cache / ws** wrap Redis for their respective jobs.
- **worker** runs background goroutines independent of any request.

---

## Tech stack

| Concern       | Choice                         | Why                                       |
| ------------- | ------------------------------ | ----------------------------------------- |
| Router        | `go-chi/chi`                   | Lightweight, composable middleware        |
| DB driver     | `jackc/pgx` (raw SQL)          | Learn SQL/relational design, no ORM magic |
| Migrations    | `golang-migrate`               | Versioned, reproducible schema            |
| Config        | `spf13/viper`                  | `.env` + env-var overrides                |
| Auth          | `golang-jwt/jwt` + `bcrypt`    | Stateless access tokens, stateful refresh |
| Cache / holds | `redis/go-redis`               | Cache-aside + atomic Lua holds            |
| Realtime      | `gorilla/websocket`            | Per-event broadcast                       |
| Tests         | `testing` + `stretchr/testify` |                                           |

Everything runs from **Docker Compose**: `api` (Go), `postgres`, `redis`.

---

## Project layout

```
cmd/
  api/            HTTP server entrypoint (wiring + graceful shutdown)
  seed/           one-shot idempotent seeder (admin + mock venues, //go:embed JSON)
internal/
  config/         viper-based config loading
  db/             pgx connection pool
  cache/          redis connection + cache-aside helper
  models/         domain structs (JSON tags)
  store/          persistence layer — one file per aggregate, all SQL here
  auth/           password hashing + JWT issue/verify
  middleware/     JWT authentication + RBAC
  handler/        one file per endpoint group
  hold/           Redis-backed atomic seat holds (Lua)
  ws/             WebSocket hub (Redis pub/sub backbone)
  worker/         background sweeper for expired holds
  respond/        consistent JSON success/error envelopes
migrations/       golang-migrate .up/.down SQL pairs
```

---

## Getting started

### Prerequisites

- Go 1.26+
- Docker + Docker Compose
- [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI (`brew install golang-migrate`)

### Setup

```bash
# 1. Config
cp .env.example .env          # then edit secrets

# 2. Start Postgres + Redis
make up

# 3. Create the schema
make migrate

# 4. Seed admin account + mock venues
make seed

# 5. Run the API
make run
```

Health check:

```bash
curl localhost:8080/health
# {"postgres":"Up & Running","redis":"Up & Running"}
```

> **Note on ports:** Postgres is published on host port **5433** (not 5432) so it
> doesn't collide with a locally-installed Postgres. Redis is on 6379.

---

## Roles & the core flow

| Role          | How they're created                      | Notes                                                                                      |
| ------------- | ---------------------------------------- | ------------------------------------------------------------------------------------------ |
| **Admin**     | **Seeded only** — never via the API      | Logs in through a separate secret route guarded by a static `adminAccessKey`               |
| **Organizer** | Self-registers, starts `pending_payment` | Pays a mock platform fee to activate; then claims a venue (admin-approved) and runs events |
| **Attendee**  | Self-registers, active immediately       | Browses events, holds seats, books                                                         |

The end-to-end journey:

1. Seed data exists at startup: mock venues (all `unclaimed`) + one admin.
2. Organizer registers → `pending_payment`.
3. Organizer pays mock fee → `active`.
4. Organizer submits a venue **registration request** with a mock document.
5. Admin approves it on the dashboard → venue becomes `claimed`.
6. Organizer creates **one** event at that venue (a second active event is rejected).
7. Organizer defines the seat map and publishes the event.
8. Attendee opens the live seat map (`available` / `held` / `booked`).
9. Attendee **holds** seats → atomic, short-lived; everyone watching gets a `seat.held` push.
10. Attendee **books** before the hold expires → seats become `booked`.
11. If the attendee walks away, the **sweeper** auto-releases the hold at TTL and broadcasts `seat.released`.
12. Attendee can cancel a booking, freeing the seats live.

---

## API reference

All endpoints are prefixed `/api/v1`. Auth column: public · auth (any logged-in
user) · role names = that role required.

### Auth & onboarding

| Method | Path                       | Auth          | Description                                                                            |
| ------ | -------------------------- | ------------- | -------------------------------------------------------------------------------------- |
| POST   | `/auth/register`           | public        | Register `attendee` (active) or `organizer` (`pending_payment`). Cannot create admins. |
| POST   | `/auth/login`              | public        | Attendee/organizer login. Pending organizers are blocked.                              |
| POST   | `/auth/admin/login`        | secret        | Body `{ email, password, adminAccessKey }`. Seeded admins only.                        |
| POST   | `/auth/refresh`            | refresh token | Rotates the token pair (old refresh token revoked).                                    |
| POST   | `/auth/organizer/activate` | organizer     | Body `{ paymentMock }`. Pays mock fee → active.                                        |
| GET    | `/users/me`                | auth          | Current profile.                                                                       |

### Venues (read-only; pre-seeded)

| Method | Path                       | Auth   | Description                              |
| ------ | -------------------------- | ------ | ---------------------------------------- |
| GET    | `/venues?status=unclaimed` | public | List venues, filterable by claim status. |
| GET    | `/venues/{venueId}`        | public | Venue detail.                            |

### Venue registration (claim workflow)

| Method | Path                                                     | Auth      | Description                                           |
| ------ | -------------------------------------------------------- | --------- | ----------------------------------------------------- |
| POST   | `/venues/{venueId}/registration-requests`                | organizer | Body `{ documentMock }`. One pending claim per venue. |
| GET    | `/venues/registration-requests/me`                       | organizer | Own requests.                                         |
| GET    | `/admin/venue-registration-requests?status=pending`      | admin     | Review queue.                                         |
| POST   | `/admin/venue-registration-requests/{requestId}/approve` | admin     | Venue → claimed, atomically.                          |
| POST   | `/admin/venue-registration-requests/{requestId}/reject`  | admin     | Body `{ reason }`.                                    |

### Events

| Method | Path                        | Auth                   | Description                                    |
| ------ | --------------------------- | ---------------------- | ---------------------------------------------- |
| POST   | `/events`                   | organizer (owns venue) | Rejected if venue already has an active event. |
| GET    | `/events?status=published`  | public                 | List events (cache-aside).                     |
| GET    | `/events/{eventId}`         | public                 | Event detail (cache-aside).                    |
| PATCH  | `/events/{eventId}`         | owner/admin            | Update.                                        |
| POST   | `/events/{eventId}/publish` | owner/admin            | Make bookable.                                 |
| POST   | `/events/{eventId}/cancel`  | owner/admin            | Cancel, frees the venue.                       |

### Seat map

| Method | Path                      | Auth   | Description                                           |
| ------ | ------------------------- | ------ | ----------------------------------------------------- |
| POST   | `/events/{eventId}/seats` | owner  | Bulk-define seats (once).                             |
| GET    | `/events/{eventId}/seats` | public | Full map + live status (`available`/`held`/`booked`). |

### Holds — the concurrency feature

| Method | Path                      | Auth             | Description                                                           |
| ------ | ------------------------- | ---------------- | --------------------------------------------------------------------- |
| POST   | `/events/{eventId}/holds` | attendee         | Body `{ seatIds }`. Atomic; returns `{ holdId, seatIds, expiresAt }`. |
| DELETE | `/holds/{holdId}`         | attendee (owner) | Manually release.                                                     |

### Bookings

| Method | Path                           | Auth        | Description                                               |
| ------ | ------------------------------ | ----------- | --------------------------------------------------------- |
| POST   | `/bookings`                    | attendee    | Body `{ holdId, paymentMock }`. Hold → permanent booking. |
| GET    | `/bookings`                    | attendee    | Own bookings.                                             |
| GET    | `/bookings/{bookingId}`        | owner/admin | Detail.                                                   |
| POST   | `/bookings/{bookingId}/cancel` | owner/admin | Cancel, free seats.                                       |

### Admin dashboard

| Method | Path              | Auth  | Description   |
| ------ | ----------------- | ----- | ------------- |
| GET    | `/admin/events`   | admin | All events.   |
| GET    | `/admin/bookings` | admin | All bookings. |
| GET    | `/admin/users`    | admin | All users.    |

### Realtime

| Protocol | Path                      | Description                                                                            |
| -------- | ------------------------- | -------------------------------------------------------------------------------------- |
| WS       | `/ws/v1/events/{eventId}` | Live `seat.held` / `seat.released` / `seat.booked` (each with `seatId` + `timestamp`). |

---

## How the hard parts work

### Concurrency-safe seat holds (the centerpiece)

Holds live entirely in **Redis** — Postgres only ever stores `available` /
`booked`. The transient `held` state is derived by overlaying Redis on top of
the DB truth when serving the seat map.

Each event has a sorted set `event:{eventId}:holds` whose **members are seat
ids** and **scores are expiry** (unix seconds). A seat is "held" iff it's in the
set with a score greater than now.

Placing a hold runs a **Lua script** that Redis executes atomically — to
completion, with nothing else interleaved:

```
for each seat: if ZSCORE > now  →  return that seat (conflict, abort)
otherwise:     ZADD every seat with the expiry score, write the hold hash
```

Because the whole check-and-set is one atomic script, two concurrent requests
for the same seat can never both pass the check. This is why an application
mutex wouldn't be enough — it wouldn't hold across multiple API instances, but a
single-threaded Redis script does.

If Redis is unreachable, hold creation **fails closed** (HTTP 503) — we never
grant a hold we can't guarantee is exclusive.

> Proven by `internal/hold/hold_test.go`: 50 goroutines race for one seat, and
> the test asserts exactly one wins.

### Booking finalization (transactional consistency)

Converting a hold to a booking happens in a single Postgres transaction:

1. `SELECT … FOR UPDATE` locks the seat rows.
2. Re-verify every seat is still `available` (defence in depth beyond the hold).
3. Flip seats → `booked`, insert the booking, the `booking_seats` join rows, and the ticket payment.
4. Commit.

Only **after** the DB commits do we consume the Redis hold and broadcast
`seat.booked`. If anything fails mid-way, the transaction rolls back and the
hold's TTL still cleans up Redis — the two stores can't end up disagreeing.

### Real-time broadcasts

The WebSocket hub fans out through **Redis pub/sub**: a seat change is published
to `ws:event:{eventId}`; every API instance is subscribed and pushes it to its
locally-connected sockets. This keeps multiple instances in sync without them
talking to each other directly.

### Background sweeper

A goroutine ticks every 10s, scans the event sorted sets for entries whose score
is `<= now` (expired), removes them, and broadcasts `seat.released`. The `ZREM`
makes it idempotent — a seat is never released or announced twice.

### Caching

Event list and detail reads are **cache-aside** in Redis (30s / 60s TTLs): read
through on a miss, served from cache on a hit, and **invalidated on any write**
(create/update/publish/cancel). The seat map is deliberately _not_ cached — its
status changes every time someone holds a seat, so a cache would fight the
real-time requirement. Knowing when _not_ to cache is part of the point.

### Auth & security

- **Passwords**: bcrypt (deliberately slow → expensive to brute-force).
- **Access tokens**: short-lived stateless JWTs (HS256), carrying user id + role.
- **Refresh tokens**: **stateful** — only a SHA-256 hash is stored, and rotation
  revokes the old one. This buys real logout/revocation that plain JWTs can't.
- **The admin route** isn't protected by obscurity. Three real checks guard it:
  admins can never be created via registration, a static `adminAccessKey` is
  required on top of the password, and the account must already exist as an admin.

---

## Data model

Eight tables, and migrations enforce the business rules **at the database level**
as a backstop to application logic:

- **One active event per venue** — partial unique index on `events(venue_id)
WHERE status NOT IN ('cancelled','completed')`.
- **One pending claim per venue** — partial unique index on
  `venue_registration_requests(venue_id) WHERE status = 'pending'`.
- **No duplicate seats** — unique `(event_id, section, seat_row, number)`.
- **A seat belongs to one booking** — unique `seat_id` on `booking_seats`.
- **Venue claim consistency** — a `CHECK` ties `claim_status` and `organizer_id`
  so they can never drift apart.
- **Role-scoped status** — a `CHECK` ensures only organizers can be `pending_payment`.
- **Payment type integrity** — a `CHECK` ties payment `type` to whether a
  `booking_id` is present.

`updated_at` is maintained by a shared Postgres trigger, not the application.

---

## Testing

```bash
make test          # all tests (Redis must be running)
go test ./internal/auth/ -count=1
go test ./internal/hold/ -count=1   # includes the concurrency proof
```

The hold tests connect to the local Redis and **skip** automatically if it isn't
running, so unit tests (`auth`) stay runnable anywhere.
