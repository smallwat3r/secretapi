package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("expected default redis URL, got %s", cfg.RedisURL)
	}
	if cfg.RedisPoolSize != 10 {
		t.Errorf("expected default pool size 10, got %d", cfg.RedisPoolSize)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear env vars
	os.Unsetenv("PORT")
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("REDIS_POOL_SIZE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	os.Setenv("PORT", "3000")
	defer os.Unsetenv("PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	os.Setenv("PORT", "not-a-number")
	defer os.Unsetenv("PORT")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestLoad_CustomRedisURL(t *testing.T) {
	os.Setenv("REDIS_URL", "redis://custom:6380/1")
	defer os.Unsetenv("REDIS_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisURL != "redis://custom:6380/1" {
		t.Errorf("expected custom redis URL, got %s", cfg.RedisURL)
	}
}

func TestLoad_CustomPoolSize(t *testing.T) {
	os.Setenv("REDIS_POOL_SIZE", "20")
	defer os.Unsetenv("REDIS_POOL_SIZE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisPoolSize != 20 {
		t.Errorf("expected pool size 20, got %d", cfg.RedisPoolSize)
	}
}

func TestLoad_InvalidPoolSize(t *testing.T) {
	testCases := []string{"0", "-1", "abc"}

	for _, val := range testCases {
		t.Run(val, func(t *testing.T) {
			os.Setenv("REDIS_POOL_SIZE", val)
			defer os.Unsetenv("REDIS_POOL_SIZE")

			_, err := Load()
			if err == nil {
				t.Error("expected error for invalid pool size")
			}
		})
	}
}

func TestLoad_CustomMinIdle(t *testing.T) {
	os.Setenv("REDIS_MIN_IDLE", "5")
	defer os.Unsetenv("REDIS_MIN_IDLE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisMinIdle != 5 {
		t.Errorf("expected min idle 5, got %d", cfg.RedisMinIdle)
	}
}

func TestLoad_InvalidMinIdle(t *testing.T) {
	os.Setenv("REDIS_MIN_IDLE", "-1")
	defer os.Unsetenv("REDIS_MIN_IDLE")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid min idle")
	}
}

func TestLoad_CustomShutdownTimeout(t *testing.T) {
	os.Setenv("SHUTDOWN_TIMEOUT", "10s")
	defer os.Unsetenv("SHUTDOWN_TIMEOUT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected shutdown timeout 10s, got %v", cfg.ShutdownTimeout)
	}
}

func TestLoad_InvalidShutdownTimeout(t *testing.T) {
	os.Setenv("SHUTDOWN_TIMEOUT", "invalid")
	defer os.Unsetenv("SHUTDOWN_TIMEOUT")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid shutdown timeout")
	}
}

func TestConfig_ListenAddr(t *testing.T) {
	cfg := Config{Port: "9000"}
	if cfg.ListenAddr() != ":9000" {
		t.Errorf("expected :9000, got %s", cfg.ListenAddr())
	}
}

func TestLoad_RequireHTTPSDefault(t *testing.T) {
	os.Unsetenv("NO_HTTPS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RequireHTTPS != true {
		t.Error("expected RequireHTTPS to default to true")
	}
}

func TestLoad_NoHTTPSDisablesRequireHTTPS(t *testing.T) {
	testCases := []string{"1", "true"}

	for _, val := range testCases {
		t.Run(val, func(t *testing.T) {
			os.Setenv("NO_HTTPS", val)
			defer os.Unsetenv("NO_HTTPS")

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.RequireHTTPS != false {
				t.Errorf("expected RequireHTTPS to be false when NO_HTTPS=%s", val)
			}
		})
	}
}

func TestLoad_NoHTTPSIgnoresOtherValues(t *testing.T) {
	testCases := []string{"0", "false", "no", ""}

	for _, val := range testCases {
		t.Run(val, func(t *testing.T) {
			os.Setenv("NO_HTTPS", val)
			defer os.Unsetenv("NO_HTTPS")

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.RequireHTTPS != true {
				t.Errorf("expected RequireHTTPS to be true when NO_HTTPS=%q", val)
			}
		})
	}
}
