// Package hold manages ephemeral, concurrency-safe seat holds in Redis.
//
// Redis is the single source of truth for the transient "held" state. Each
// event has a sorted set  event:{eventId}:holds  whose members are seatIds and
// whose scores are the hold's expiry (unix seconds). A seat is "currently held"
// iff it appears in that set with a score greater than now. A per-hold Redis
// HASH  hold:{holdId}  stores the hold's metadata and carries the TTL.
//
// All multi-seat mutations run as Lua scripts so the check-and-set is atomic:
// two concurrent requests for the same seat can never both succeed, because
// Redis executes a script to completion before running anything else.
package hold

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrSeatTaken means at least one requested seat was already held.
	ErrSeatTaken = errors.New("seat already held")
	// ErrHoldNotFound means the hold expired or never existed.
	ErrHoldNotFound = errors.New("hold not found")
	// ErrNotOwner means the caller doesn't own the hold they're releasing.
	ErrNotOwner = errors.New("not the hold owner")
)

type Manager struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *Manager {
	return &Manager{rdb: rdb, ttl: ttl}
}

// Hold is the metadata describing one active hold.
type Hold struct {
	ID        string    `json:"holdId"`
	EventID   string    `json:"eventId"`
	UserID    string    `json:"userId"`
	SeatIDs   []string  `json:"seatIds"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Released reports the seats freed for a single event (used to broadcast).
type Released struct {
	EventID string
	SeatIDs []string
}

func zsetKey(eventID string) string  { return "event:" + eventID + ":holds" }
func recordKey(holdID string) string { return "hold:" + holdID }

// createScript atomically verifies every seat is free, then claims them all.
// Returns "OK" on success, or the id of the first conflicting seat.
//
//	KEYS[1] = event zset      KEYS[2] = hold record hash
//	ARGV[1] = now  ARGV[2] = expiresAt  ARGV[3] = ttl
//	ARGV[4] = userId  ARGV[5] = eventId  ARGV[6..] = seatIds
var createScript = redis.NewScript(`
local now = tonumber(ARGV[1])
for i = 6, #ARGV do
  local sc = redis.call('ZSCORE', KEYS[1], ARGV[i])
  if sc and tonumber(sc) > now then
    return ARGV[i]
  end
end
local seats = {}
for i = 6, #ARGV do
  redis.call('ZADD', KEYS[1], ARGV[2], ARGV[i])
  table.insert(seats, ARGV[i])
end
redis.call('HSET', KEYS[2], 'eventId', ARGV[5], 'userId', ARGV[4], 'expiresAt', ARGV[2], 'seats', table.concat(seats, ','))
redis.call('EXPIRE', KEYS[2], tonumber(ARGV[3]))
return 'OK'
`)

// releaseScript removes a hold and its seats from the event zset.
// ARGV[1] = expectedUserId ("" skips the ownership check, used at booking time)
// Returns {status, eventId, seatsCsv} where status is OK / MISSING / FORBIDDEN.
var releaseScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then
  return {'MISSING'}
end
local userId = redis.call('HGET', KEYS[1], 'userId')
if ARGV[1] ~= '' and userId ~= ARGV[1] then
  return {'FORBIDDEN'}
end
local eventId = redis.call('HGET', KEYS[1], 'eventId')
local seats = redis.call('HGET', KEYS[1], 'seats')
local zkey = 'event:' .. eventId .. ':holds'
for seatId in string.gmatch(seats, '([^,]+)') do
  redis.call('ZREM', zkey, seatId)
end
redis.call('DEL', KEYS[1])
return {'OK', eventId, seats}
`)

// Create places an atomic hold on the given seats for the user.
func (m *Manager) Create(ctx context.Context, eventID, userID string, seatIDs []string) (*Hold, error) {
	holdID := uuid.NewString()
	now := time.Now()
	expiresAt := now.Add(m.ttl)

	argv := []any{
		now.Unix(),
		expiresAt.Unix(),
		int(m.ttl.Seconds()),
		userID,
		eventID,
	}
	for _, s := range seatIDs {
		argv = append(argv, s)
	}

	res, err := createScript.Run(ctx, m.rdb,
		[]string{zsetKey(eventID), recordKey(holdID)}, argv...).Result()
	if err != nil {
		return nil, err
	}
	if res != "OK" {
		return nil, ErrSeatTaken
	}

	return &Hold{
		ID:        holdID,
		EventID:   eventID,
		UserID:    userID,
		SeatIDs:   seatIDs,
		ExpiresAt: expiresAt,
	}, nil
}

// Get returns hold metadata, or ErrHoldNotFound if it expired.
func (m *Manager) Get(ctx context.Context, holdID string) (*Hold, error) {
	vals, err := m.rdb.HGetAll(ctx, recordKey(holdID)).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, ErrHoldNotFound
	}
	return &Hold{
		ID:        holdID,
		EventID:   vals["eventId"],
		UserID:    vals["userId"],
		SeatIDs:   strings.Split(vals["seats"], ","),
		ExpiresAt: parseUnix(vals["expiresAt"]),
	}, nil
}

// Release frees a hold the user owns (manual DELETE /holds/{id}).
func (m *Manager) Release(ctx context.Context, holdID, userID string) (*Released, error) {
	return m.release(ctx, holdID, userID)
}

// Consume frees a hold without an ownership check, used after a booking has
// already been validated and persisted.
func (m *Manager) Consume(ctx context.Context, holdID string) (*Released, error) {
	return m.release(ctx, holdID, "")
}

func (m *Manager) release(ctx context.Context, holdID, expectedUser string) (*Released, error) {
	raw, err := releaseScript.Run(ctx, m.rdb, []string{recordKey(holdID)}, expectedUser).Result()
	if err != nil {
		return nil, err
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil, ErrHoldNotFound
	}
	switch arr[0] {
	case "MISSING":
		return nil, ErrHoldNotFound
	case "FORBIDDEN":
		return nil, ErrNotOwner
	}
	eventID, _ := arr[1].(string)
	seatsCsv, _ := arr[2].(string)
	return &Released{EventID: eventID, SeatIDs: strings.Split(seatsCsv, ",")}, nil
}

// HeldSeats returns the ids of seats currently held for an event (score > now).
// Stale expired entries that the sweeper hasn't removed yet are excluded by the
// score filter, so this is always accurate.
func (m *Manager) HeldSeats(ctx context.Context, eventID string) ([]string, error) {
	now := time.Now().Unix()
	return m.rdb.ZRangeByScore(ctx, zsetKey(eventID), &redis.ZRangeBy{
		Min: "(" + itoa(now), // exclusive: strictly greater than now
		Max: "+inf",
	}).Result()
}

// SweepExpired finds every expired hold across all events, removes the expired
// seat entries from their zsets, and returns what was freed so the caller can
// broadcast seat.released. ZREM makes this idempotent — a seat freed once won't
// be reported again.
func (m *Manager) SweepExpired(ctx context.Context) ([]Released, error) {
	now := time.Now().Unix()
	var released []Released

	var cursor uint64
	for {
		keys, next, err := m.rdb.Scan(ctx, cursor, "event:*:holds", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, zkey := range keys {
			expired, err := m.rdb.ZRangeByScore(ctx, zkey, &redis.ZRangeBy{
				Min: "-inf",
				Max: itoa(now), // inclusive: <= now is expired
			}).Result()
			if err != nil {
				return nil, err
			}
			if len(expired) == 0 {
				continue
			}
			// Remove them so we don't broadcast the same release twice.
			members := make([]any, len(expired))
			for i, s := range expired {
				members[i] = s
			}
			if err := m.rdb.ZRem(ctx, zkey, members...).Err(); err != nil {
				return nil, err
			}
			eventID := strings.TrimSuffix(strings.TrimPrefix(zkey, "event:"), ":holds")
			released = append(released, Released{EventID: eventID, SeatIDs: expired})
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return released, nil
}
