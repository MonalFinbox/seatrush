package cache

import (
    "context"
    "fmt"

    "github.com/redis/go-redis/v9"
)

func New(addr string) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr: addr,
    })

    if err := client.Ping(context.Background()).Err(); err != nil {
        return nil, fmt.Errorf("could not ping redis: %w", err)
    }

    return client, nil
}