# SeatRush — Frontend

React single-page app for the [SeatRush](../seatrush) ticket-booking backend.
Three role experiences (attendee, organizer, admin) over the locked `/api/v1`
contract, with a live, WebSocket-driven seat map.

## Stack

- **React 18 + TypeScript** (Vite)
- **Tailwind CSS** + **shadcn/ui** (Radix primitives, components live in `src/components/ui`)
- **axios** — API client with a token-refresh interceptor
- **zod** — runtime validation of every API response and all forms
- **react-router-dom** — routing + role-gated routes
- **sonner** — toasts

## Getting started

The backend must be running first (see `../seatrush`).

```bash
npm install
cp .env .env.local   # optional; defaults point at localhost:8080
npm run dev          # http://localhost:5173
```

`.env`:

```
VITE_API_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/ws/v1
```

> The backend allows the dev origin `http://localhost:5173` via CORS
> (`CORS_ORIGINS` in the backend `.env`).

Build for production:

```bash
npm run build && npm run preview
```

## Project layout

```
src/
  lib/
    api.ts          axios instance, token storage, refresh-on-401 interceptor
    services.ts     typed API calls, one object per resource
    utils.ts        cn() + formatters
  types/schemas.ts  zod schemas + inferred types (mirror the backend JSON)
  store/auth.tsx    auth context (login/logout, hydrate /users/me)
  hooks/
    useEventSocket  subscribe to live seat events for one event
  components/
    ui/             shadcn primitives
    layout/         Navbar, route guards, StatusBadge
    SeatMap.tsx     interactive seat grid
  pages/
    Home, Login, Register, AdminLogin, EventDetail, MyBookings
    organizer/      OrganizerDashboard, EventManage
    admin/          AdminDashboard
  App.tsx           router + layout
```

## How auth works

- On login/register the `{ accessToken, refreshToken }` pair is stored in
  `localStorage`; the user object hydrates React state.
- Every request attaches `Authorization: Bearer <access>`.
- On a `401`, the axios interceptor transparently calls `/auth/refresh` once,
  retries the original request, and — if refresh fails — clears the session and
  bounces to `/login`.
- Routes are gated by role with `<ProtectedRoute roles={[...]} />`.
- `/admin/login` is an unlinked route requiring the static admin access key.

## The live seat map

`EventDetail` opens a WebSocket to `/ws/v1/events/{eventId}` and applies
`seat.held` / `seat.released` / `seat.booked` messages to the grid in real time.
An attendee selects available seats → **holds** them (atomic on the server) →
sees a countdown → **pays (mock) & confirms**. Everyone watching the event sees
the seat states change instantly.

## Role journeys

- **Attendee** — browse events → open seat map → hold → book → manage bookings.
- **Organizer** — register → pay mock fee to activate → claim a venue (with a
  mock document) → after admin approval, create an event → build the seat map →
  publish → cancel.
- **Admin** — sign in at `/admin/login` → review the claim queue
  (approve/reject) → inspect all events, bookings, and users.
