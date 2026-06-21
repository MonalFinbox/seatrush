package hold

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// testRedis connects to the local Redis, skipping the test if it isn't running.
func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	c := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := c.Ping(context.Background()).Err(); err != nil {
		t.Skip("redis not available on localhost:6379, skipping")
	}
	return c
}

// TestConcurrentHoldsSingleWinner is the centerpiece guarantee: when many
// requests race for the same seat, exactly one succeeds and the rest fail.
func TestConcurrentHoldsSingleWinner(t *testing.T) {
	rdb := testRedis(t)
	m := New(rdb, 30*time.Second)
	ctx := context.Background()

	eventID := "test-" + uuid.NewString()
	seat := uuid.NewString()
	defer rdb.Del(ctx, zsetKey(eventID))

	const racers = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0

	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := m.Create(ctx, eventID, uuid.NewString(), []string{seat}); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	require.Equal(t, 1, wins, "exactly one of %d concurrent holds must win", racers)
}

// TestReleaseFreesSeat verifies a released seat can be held again.
func TestReleaseFreesSeat(t *testing.T) {
	rdb := testRedis(t)
	m := New(rdb, 30*time.Second)
	ctx := context.Background()

	eventID := "test-" + uuid.NewString()
	seat := uuid.NewString()
	user := uuid.NewString()
	defer rdb.Del(ctx, zsetKey(eventID))

	h1, err := m.Create(ctx, eventID, user, []string{seat})
	require.NoError(t, err)

	// A different user can't hold the same seat.
	_, err = m.Create(ctx, eventID, uuid.NewString(), []string{seat})
	require.ErrorIs(t, err, ErrSeatTaken)

	// Release, then the seat becomes holdable again.
	_, err = m.Release(ctx, h1.ID, user)
	require.NoError(t, err)

	_, err = m.Create(ctx, eventID, uuid.NewString(), []string{seat})
	require.NoError(t, err)
}

// TestReleaseWrongOwnerRejected verifies ownership is enforced.
func TestReleaseWrongOwnerRejected(t *testing.T) {
	rdb := testRedis(t)
	m := New(rdb, 30*time.Second)
	ctx := context.Background()

	eventID := "test-" + uuid.NewString()
	seat := uuid.NewString()
	owner := uuid.NewString()
	defer rdb.Del(ctx, zsetKey(eventID))

	h, err := m.Create(ctx, eventID, owner, []string{seat})
	require.NoError(t, err)

	_, err = m.Release(ctx, h.ID, "someone-else")
	require.ErrorIs(t, err, ErrNotOwner)
}
