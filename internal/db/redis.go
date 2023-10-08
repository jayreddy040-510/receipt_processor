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

func (rs *RedisStore) CheckConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(rs.config.DbTimeoutInMs))
	defer cancel()

	return rs.client.Ping(ctx).Err()
}

func (rs *RedisStore) GetKey(key string) (string, error) {
	// see design decision in setKey below
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(rs.config.DbTimeoutInMs))
	defer cancel()

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

func (rs *RedisStore) SetKey(key, value string) error {
	// design decision: pass in ctx with timeout to setter from main or define here?
	// because only 1 way we plan on setting and don't plan on changing cancel/timeout logic,
	// i think it's fine to init ctx here
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(rs.config.DbTimeoutInMs))
	defer cancel()

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
