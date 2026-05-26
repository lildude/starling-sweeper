// Package cache implements a Redis cache.
package cache

import (
	"context"
	"encoding/json"
	"errors"

	redis "github.com/go-redis/redis/v8"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any) error
	GetJSON(ctx context.Context, key string, value any) error
	SetJSON(ctx context.Context, key string, value any) error
}

type RedisCache struct {
	conn *redis.Client
}

func NewRedisCache(ctx context.Context, addr string) (Cache, error) {
	opt, err := redis.ParseURL(addr)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &RedisCache{
		conn: client,
	}, nil
}

// Set stores a value in the cache.
func (rc *RedisCache) Set(ctx context.Context, key string, value any) error {
	return rc.conn.Set(ctx, key, value, 0).Err()
}

// Get retrieves a value from the cache.
func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
	value, err := rc.conn.Get(ctx, key).Result()
	if err == nil || errors.Is(err, redis.Nil) {
		return value, nil
	}

	return "", err
}

// GetJSON retrieves a JSON string and unmarshals it into the given interface.
func (rc *RedisCache) GetJSON(ctx context.Context, key string, value any) error {
	v, err := rc.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(v), &value); err != nil {
		return err
	}
	return nil
}

// SetJSON stores a struct as a JSON string.
func (rc *RedisCache) SetJSON(ctx context.Context, key string, value any) error {
	t, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return rc.Set(ctx, key, string(t))
}
