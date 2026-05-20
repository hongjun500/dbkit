package dbkit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient 对 go-redis 的薄封装，提供常用操作与 Pub/Sub 入口。
type RedisClient struct {
	rdb *redis.Client
}

func openRedis(ctx context.Context, cfg RedisConfig, log Logger) (*RedisClient, error) {
	cfg = cfg.withDefaults()
	if cfg.Addr == "" {
		return nil, fmt.Errorf("dbkit redis: addr is required when enabled")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.Dial,
		ReadTimeout:  cfg.Read,
		WriteTimeout: cfg.Write,
	})

	pingCtx, cancel := context.WithTimeout(ctx, cfg.Dial)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("dbkit redis: ping: %w", err)
	}

	log.Info("redis connected", String("component", "redis"), String("addr", cfg.Addr))
	return &RedisClient{rdb: rdb}, nil
}

func (c *RedisClient) Raw() *redis.Client { return c.rdb }

func (c *RedisClient) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *RedisClient) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *RedisClient) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

func (c *RedisClient) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.rdb.Exists(ctx, keys...).Result()
}

// Publish 发布消息到频道。
func (c *RedisClient) Publish(ctx context.Context, channel string, message any) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

// Subscribe 返回 Pub/Sub 订阅，调用方负责 Close。
func (c *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channels...)
}
