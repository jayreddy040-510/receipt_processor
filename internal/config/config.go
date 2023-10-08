package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort       string
	RedisAddr        string
	DbTimeoutInMs    time.Duration
	RedisTTLInSec    time.Duration
	MaxDBConnRetries int
}

func Load() (Config, error) {
	// design decision: return Config or *Config? since main functionality of Config is
	// to read it and not write to it, decided to return struct
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	// strconv will throw error if os.Getenv("FOO") returns "" - can catch early
	dbTimeoutInMs, err := strconv.Atoi(os.Getenv("DB_TIMEOUT_IN_MS"))
	if err != nil {
		return Config{}, fmt.Errorf("Error converting DB_TIMEOUT env to int: %v", err)
	}

	redisTTLInSec, err := strconv.Atoi(os.Getenv("REDIS_TTL_IN_S"))
	if err != nil {
		return Config{}, fmt.Errorf("Error converting REDIS_TTL env to int: %v", err)
	}

	maxDBConnRetries, err := strconv.Atoi(os.Getenv("MAX_DB_CONN_RETRIES"))
	if err != nil {
		return Config{}, fmt.Errorf("Error converting MAX_DB_CONN_RETRIES env to int: %v", err)
	}

	appConfig := Config{
		ServerPort:       serverPort,
		RedisAddr:        redisAddr,
		DbTimeoutInMs:    time.Millisecond * time.Duration(dbTimeoutInMs),
		RedisTTLInSec:    time.Second * time.Duration(redisTTLInSec),
		MaxDBConnRetries: maxDBConnRetries,
	}
	return appConfig, nil
}
