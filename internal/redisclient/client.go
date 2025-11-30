package redisclient

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisService interface {
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Set(ctx context.Context, key string, value any, exp int) error
	SetRange(ctx context.Context, key string, offset int64, value any) error
	Ping(ctx context.Context) error
	Close() error
}

type Client struct {
	rdb *redis.Client
}

func (c *Client) Set(ctx context.Context, key string, value any, exp int) error {
	var expire time.Duration

	if exp > 0 {
		expire = time.Duration(exp) * time.Second
	}
	return c.rdb.Set(ctx, key, value, expire).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.rdb.Get(ctx, key).Result()

	if err == redis.Nil {
		return "", err
	}

	return value, err
}

func (c *Client) SetRange(ctx context.Context, key string, offset int64, value any) error {
	return c.rdb.SetRange(ctx, key, offset, value.(string)).Err()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Del(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

func NewClient() (*Client, error) {
	addr := os.Getenv("REDIS_ENDPOINT")

	if addr == "" {
		addr = "0.0.0.0:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	c := &Client{rdb: rdb}

	if err := c.Ping(context.Background()); err != nil {
		return nil, err
	}

	return c, nil
}
