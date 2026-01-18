package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Server settings
	Port              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int

	// Redis settings
	RedisURL          string
	RedisPoolSize     int
	RedisMinIdle      int
	RedisDialTimeout  time.Duration
	RedisReadTimeout  time.Duration
	RedisWriteTimeout time.Duration
	RedisPoolTimeout  time.Duration

	// Shutdown settings
	ShutdownTimeout time.Duration

	// Security settings
	RequireHTTPS bool // enforce HTTPS with HSTS header (disable with NO_HTTPS=1)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Port:              "8080",
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB

		RedisURL:          "redis://localhost:6379/0",
		RedisPoolSize:     10,
		RedisMinIdle:      2,
		RedisDialTimeout:  5 * time.Second,
		RedisReadTimeout:  3 * time.Second,
		RedisWriteTimeout: 3 * time.Second,
		RedisPoolTimeout:  4 * time.Second,

		ShutdownTimeout: 5 * time.Second,

		RequireHTTPS: true, // secure default: enforce HTTPS
	}
}

// Load reads configuration from environment variables and validates it.
func Load() (Config, error) {
	cfg := DefaultConfig()

	// Server settings
	if port := os.Getenv("PORT"); port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return Config{}, fmt.Errorf("PORT must be a valid number: %w", err)
		}
		cfg.Port = port
	}

	// Redis settings
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.RedisURL = redisURL
	}

	if poolSize := os.Getenv("REDIS_POOL_SIZE"); poolSize != "" {
		size, err := strconv.Atoi(poolSize)
		if err != nil || size < 1 {
			return Config{}, errors.New("REDIS_POOL_SIZE must be a positive integer")
		}
		cfg.RedisPoolSize = size
	}

	if minIdle := os.Getenv("REDIS_MIN_IDLE"); minIdle != "" {
		idle, err := strconv.Atoi(minIdle)
		if err != nil || idle < 0 {
			return Config{}, errors.New("REDIS_MIN_IDLE must be a non-negative integer")
		}
		cfg.RedisMinIdle = idle
	}

	// Shutdown settings
	if timeout := os.Getenv("SHUTDOWN_TIMEOUT"); timeout != "" {
		dur, err := time.ParseDuration(timeout)
		if err != nil {
			return Config{}, fmt.Errorf(
				"SHUTDOWN_TIMEOUT must be a valid duration: %w", err)
		}
		cfg.ShutdownTimeout = dur
	}

	// Security settings
	if noHTTPS := os.Getenv("NO_HTTPS"); noHTTPS == "1" || noHTTPS == "true" {
		cfg.RequireHTTPS = false
	}

	return cfg, nil
}

// ListenAddr returns the address string for the HTTP server.
func (c Config) ListenAddr() string {
	return ":" + c.Port
}
