package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// Config holds every setting that can be provided through the environment.
type Config struct {
	Port string

	RedisURL      string
	RedisPoolSize int
	RedisMinIdle  int

	ShutdownTimeout time.Duration

	RequireHTTPS     bool   // enforce HTTPS with HSTS header (disable with NO_HTTPS=1)
	CanonicalHost    string // canonical hostname for HTTPS redirects (CANONICAL_HOST)
	TrustedProxyCIDR string // CIDR from which proxy headers are trusted (TRUSTED_PROXY_CIDR)

	DefaultTheme string // "" | "light" | "dark"
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Port:            "8080",
		RedisURL:        "redis://localhost:6379/0",
		RedisPoolSize:   10,
		RedisMinIdle:    2,
		ShutdownTimeout: 5 * time.Second,
		RequireHTTPS:    true, // secure default: enforce HTTPS
	}
}

// Load reads configuration from environment variables and validates it.
func Load() (Config, error) {
	cfg := DefaultConfig()

	if port := os.Getenv("PORT"); port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return Config{}, fmt.Errorf("PORT must be a valid number: %w", err)
		}
		cfg.Port = port
	}

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

	if timeout := os.Getenv("SHUTDOWN_TIMEOUT"); timeout != "" {
		dur, err := time.ParseDuration(timeout)
		if err != nil {
			return Config{}, fmt.Errorf(
				"SHUTDOWN_TIMEOUT must be a valid duration: %w", err)
		}
		cfg.ShutdownTimeout = dur
	}

	if noHTTPS := os.Getenv("NO_HTTPS"); noHTTPS == "1" || noHTTPS == "true" {
		cfg.RequireHTTPS = false
	}

	if canonicalHost := os.Getenv("CANONICAL_HOST"); canonicalHost != "" {
		cfg.CanonicalHost = canonicalHost
	}

	if cidr := os.Getenv("TRUSTED_PROXY_CIDR"); cidr != "" {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return Config{}, fmt.Errorf("TRUSTED_PROXY_CIDR must be a valid CIDR: %w", err)
		}
		cfg.TrustedProxyCIDR = cidr
	}

	if theme := os.Getenv("DEFAULT_THEME"); theme != "" {
		if theme != "light" && theme != "dark" {
			return Config{}, fmt.Errorf("DEFAULT_THEME must be 'light' or 'dark', got %q", theme)
		}
		cfg.DefaultTheme = theme
	}

	return cfg, nil
}

// ListenAddr returns the address string for the HTTP server.
func (c Config) ListenAddr() string {
	return ":" + c.Port
}
