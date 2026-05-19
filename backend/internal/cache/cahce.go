package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	GetKeyVal(ctx context.Context, key string) (string, error)
	SetKeyValWithTTL(ctx context.Context, key string, val interface{}, ttl time.Duration) error
	Close() error
}

type RedisCache struct {
	r *redis.Client
}

func NewCache(host, port string) (Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &RedisCache{
		rdb,
	}, nil

}

func (cache *RedisCache) GetKeyVal(ctx context.Context, key string) (string, error) {
	cmd := cache.r.Get(ctx, key)
	err := cmd.Err()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return cmd.Val(), nil

}

func (cache *RedisCache) SetKeyValWithTTL(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	return cache.r.Set(ctx, key, val, ttl).Err()
}

func (cache *RedisCache) Close() error {
	return cache.r.Close()
}
