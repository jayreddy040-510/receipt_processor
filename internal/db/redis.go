package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jayreddy040-510/receipt_processor/internal/config"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	config config.Config
}

func NewRedisStore(config config.Config) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr: config.RedisAddr,
		}),
		config: config,
	}
}

func (rs *RedisStore) CheckConnection(ctx context.Context) error {
	return rs.client.Ping(ctx).Err()
}

func (rs *RedisStore) GetKey(ctx context.Context, key string) (string, error) {
	for i := 0; i < rs.config.MaxDBConnRetries; i++ {
		storedValue, err := rs.client.Get(ctx, key).Result()
		if err == context.DeadlineExceeded {
			log.Printf("Connection to DB timed out, attempting retry, retries attempted: %v", i)
			continue
		} else if err == redis.Nil {
			return "", fmt.Errorf("Key does not exist in database: %v", err)
		} else if err != nil {
			return "", fmt.Errorf("Error getting key from database: %v", err)
		} else {
			return storedValue, nil
		}
	}
	return "", fmt.Errorf("Error connecting to DB: %v. Max retries attempted.", context.DeadlineExceeded)
}

func (rs *RedisStore) SetKey(ctx context.Context, key, value string) error {
	for i := 0; i < rs.config.MaxDBConnRetries; i++ {
		err := rs.client.Set(ctx, key, value, time.Second*time.Duration(rs.config.RedisTTLInSec)).Err()
		if err == context.DeadlineExceeded {
			log.Printf("Connection to DB timed out, attempting retry, retries attempted: %v", i)
			continue
		} else if err != nil {
			return fmt.Errorf("Error setting key in database: %v", err)
		} else {
			return err
		}
	}
	return fmt.Errorf("Error connecting to DB: %v. Max retries attempted.", context.DeadlineExceeded)
}
