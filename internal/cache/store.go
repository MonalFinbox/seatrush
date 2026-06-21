package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is a thin cache-aside helper over Redis: read-through with GetJSON,
// write on miss with SetJSON, and bust with Del on writes.
type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

// GetJSON loads key into dst. found=false on a cache miss (not an error).
func (c *Cache) GetJSON(ctx context.Context, key string, dst any) (found bool, err error) {
	b, err := c.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return false, err
	}
	return true, nil
}

// SetJSON stores val as JSON under key with a TTL.
func (c *Cache) SetJSON(ctx context.Context, key string, val any, ttl time.Duration) {
	b, err := json.Marshal(val)
	if err != nil {
		return
	}
	if err := c.rdb.Set(ctx, key, b, ttl).Err(); err != nil {
		log.Printf("cache: set %s failed: %v", key, err)
	}
}

// Del removes keys (cache invalidation on writes). Missing keys are ignored.
func (c *Cache) Del(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		log.Printf("cache: del failed: %v", err)
	}
}
