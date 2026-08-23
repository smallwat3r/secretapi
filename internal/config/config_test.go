package config

import (
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
	t.Setenv("PORT", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_POOL_SIZE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("PORT", "3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestLoad_CustomRedisURL(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://custom:6380/1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisURL != "redis://custom:6380/1" {
		t.Errorf("expected custom redis URL, got %s", cfg.RedisURL)
	}
}

func TestLoad_CustomPoolSize(t *testing.T) {
	t.Setenv("REDIS_POOL_SIZE", "20")

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
			t.Setenv("REDIS_POOL_SIZE", val)

			_, err := Load()
			if err == nil {
				t.Error("expected error for invalid pool size")
			}
		})
	}
}

func TestLoad_CustomMinIdle(t *testing.T) {
	t.Setenv("REDIS_MIN_IDLE", "5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisMinIdle != 5 {
		t.Errorf("expected min idle 5, got %d", cfg.RedisMinIdle)
	}
}

func TestLoad_InvalidMinIdle(t *testing.T) {
	t.Setenv("REDIS_MIN_IDLE", "-1")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid min idle")
	}
}

func TestLoad_CustomShutdownTimeout(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT", "10s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected shutdown timeout 10s, got %v", cfg.ShutdownTimeout)
	}
}

func TestLoad_InvalidShutdownTimeout(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT", "invalid")

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
	t.Setenv("NO_HTTPS", "")

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
			t.Setenv("NO_HTTPS", val)

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

func TestLoad_DefaultTheme(t *testing.T) {
	t.Run("unset defaults to empty string", func(t *testing.T) {
		t.Setenv("DEFAULT_THEME", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.DefaultTheme != "" {
			t.Errorf("expected empty DefaultTheme, got %q", cfg.DefaultTheme)
		}
	})

	for _, theme := range []string{"light", "dark"} {
		t.Run(theme, func(t *testing.T) {
			t.Setenv("DEFAULT_THEME", theme)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.DefaultTheme != theme {
				t.Errorf("expected DefaultTheme %q, got %q", theme, cfg.DefaultTheme)
			}
		})
	}

	t.Run("invalid value returns error", func(t *testing.T) {
		t.Setenv("DEFAULT_THEME", "blue")

		_, err := Load()
		if err == nil {
			t.Error("expected error for invalid DEFAULT_THEME")
		}
	})
}

func TestLoad_NoHTTPSIgnoresOtherValues(t *testing.T) {
	testCases := []string{"0", "false", "no", ""}

	for _, val := range testCases {
		t.Run(val, func(t *testing.T) {
			t.Setenv("NO_HTTPS", val)

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

func TestLoad_TrustedProxyCIDRDefault(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.TrustedProxyCIDR != "" {
		t.Errorf("expected empty TrustedProxyCIDR by default, got %q", cfg.TrustedProxyCIDR)
	}
}

func TestLoad_TrustedProxyCIDR(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDR", "10.0.0.0/8")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.TrustedProxyCIDR != "10.0.0.0/8" {
		t.Errorf("expected TrustedProxyCIDR %q, got %q", "10.0.0.0/8", cfg.TrustedProxyCIDR)
	}
}

func TestLoad_TrustedProxyCIDRInvalid(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDR", "not-a-cidr")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid TRUSTED_PROXY_CIDR")
	}
}

func TestLoad_CanonicalHostDefault(t *testing.T) {
	t.Setenv("CANONICAL_HOST", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.CanonicalHost != "" {
		t.Errorf("expected empty CanonicalHost by default, got %q", cfg.CanonicalHost)
	}
}

func TestLoad_CanonicalHost(t *testing.T) {
	t.Setenv("CANONICAL_HOST", "secretapi.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.CanonicalHost != "secretapi.example.com" {
		t.Errorf("expected CanonicalHost %q, got %q", "secretapi.example.com", cfg.CanonicalHost)
	}
}
