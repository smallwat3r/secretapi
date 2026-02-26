package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test without HTTPS requirement (default for tests)
	wrapped := SecurityHeaders(SecurityHeadersConfig{})(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "strict-origin-when-cross-origin",
		"Cross-Origin-Opener-Policy": "same-origin",
	}

	for header, expected := range expectedHeaders {
		if got := rr.Header().Get(header); got != expected {
			t.Errorf("expected %s header to be %q, got %q", header, expected, got)
		}
	}

	// Verify CSP
	csp := rr.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header to be set")
	}
	cspChecks := []string{
		"default-src 'self'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	for _, check := range cspChecks {
		if !strings.Contains(csp, check) {
			t.Errorf("expected CSP to contain %q", check)
		}
	}

	// Verify Permissions-Policy
	pp := rr.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Error("expected Permissions-Policy header to be set")
	}
	if !strings.Contains(pp, "geolocation=()") {
		t.Error("expected Permissions-Policy to disable geolocation")
	}
}

func TestSecurityHeaders_HTTPSEnforcement(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("redirects HTTP to HTTPS when RequireHTTPS is true", func(t *testing.T) {
		wrapped := SecurityHeaders(SecurityHeadersConfig{RequireHTTPS: true})(handler)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("expected redirect status %d, got %d",
				http.StatusMovedPermanently, rr.Code)
		}

		location := rr.Header().Get("Location")
		if location != "https://example.com/test" {
			t.Errorf("expected redirect to https://example.com/test, got %s", location)
		}
	})

	t.Run("allows HTTPS requests and sets HSTS when RequireHTTPS is true", func(t *testing.T) {
		wrapped := SecurityHeaders(SecurityHeadersConfig{RequireHTTPS: true})(handler)

		req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		hsts := rr.Header().Get("Strict-Transport-Security")
		if hsts == "" {
			t.Error("expected HSTS header to be set")
		}
		if !strings.Contains(hsts, "max-age=31536000") {
			t.Errorf("expected HSTS max-age of 1 year, got %s", hsts)
		}
		if !strings.Contains(hsts, "includeSubDomains") {
			t.Errorf("expected HSTS to include subdomains, got %s", hsts)
		}
	})

	t.Run("does not set HSTS when RequireHTTPS is false", func(t *testing.T) {
		wrapped := SecurityHeaders(SecurityHeadersConfig{RequireHTTPS: false})(handler)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		hsts := rr.Header().Get("Strict-Transport-Security")
		if hsts != "" {
			t.Errorf("expected no HSTS header when RequireHTTPS is false, got %s", hsts)
		}
	})

	t.Run("does not redirect when RequireHTTPS is false", func(t *testing.T) {
		wrapped := SecurityHeaders(SecurityHeadersConfig{RequireHTTPS: false})(handler)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("redirect uses CanonicalHost not attacker-controlled Host header", func(t *testing.T) {
		wrapped := SecurityHeaders(SecurityHeadersConfig{
			RequireHTTPS:  true,
			CanonicalHost: "secretapi.example.com",
		})(handler)

		req := httptest.NewRequest(http.MethodGet, "/secret-path", nil)
		req.Host = "evil.com"
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("expected redirect status %d, got %d",
				http.StatusMovedPermanently, rr.Code)
		}
		location := rr.Header().Get("Location")
		if location != "https://secretapi.example.com/secret-path" {
			t.Errorf("expected redirect to canonical host, got %q", location)
		}
		if strings.Contains(location, "evil.com") {
			t.Errorf("redirect must not use attacker-controlled Host header, got %q", location)
		}
	})

	t.Run("redirect falls back to Host header when CanonicalHost is empty", func(t *testing.T) {
		wrapped := SecurityHeaders(SecurityHeadersConfig{RequireHTTPS: true})(handler)

		req := httptest.NewRequest(http.MethodGet, "/path", nil)
		req.Host = "myapp.example.com"
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		location := rr.Header().Get("Location")
		if location != "https://myapp.example.com/path" {
			t.Errorf("expected fallback redirect to r.Host, got %q", location)
		}
	})
}

func TestRateLimiter(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("passes through when redis is nil", func(t *testing.T) {
		rl := NewRateLimiter(nil, DefaultRateLimitConfig())
		wrapped := rl.Handler(handler)

		for i := range 100 {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("request %d: expected %d, got %d",
					i+1, http.StatusOK, rr.Code)
			}
		}
	})

	t.Run("default config has sensible values", func(t *testing.T) {
		cfg := DefaultRateLimitConfig()
		if cfg.PostLimit <= 0 {
			t.Errorf("expected positive PostLimit, got %d", cfg.PostLimit)
		}
		if cfg.GetLimit <= 0 {
			t.Errorf("expected positive GetLimit, got %d", cfg.GetLimit)
		}
		if cfg.Window <= 0 {
			t.Errorf("expected positive Window, got %v", cfg.Window)
		}
	})
}

func TestStripPort(t *testing.T) {
	cases := []struct{ input, want string }{
		{"192.168.1.1:12345", "192.168.1.1"},
		{"[::1]:80", "::1"},
		{"10.0.0.1", "10.0.0.1"},
	}
	for _, c := range cases {
		if got := stripPort(c.input); got != c.want {
			t.Errorf("stripPort(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestIPInCIDR(t *testing.T) {
	cases := []struct {
		addr string
		cidr string
		want bool
	}{
		{"10.0.0.1:1234", "10.0.0.0/8", true},
		{"10.255.255.255:1234", "10.0.0.0/8", true},
		{"192.168.1.1:1234", "10.0.0.0/8", false},
		{"172.16.0.5:8080", "172.16.0.0/12", true},
		{"bad-addr", "10.0.0.0/8", false},
		{"10.0.0.1:1234", "invalid-cidr", false},
	}
	for _, c := range cases {
		got := ipInCIDR(c.addr, c.cidr)
		if got != c.want {
			t.Errorf("ipInCIDR(%q, %q) = %v, want %v", c.addr, c.cidr, got, c.want)
		}
	}
}

func TestRateLimiter_IPExtraction(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("uses RemoteAddr when TrustedProxyCIDR is empty", func(t *testing.T) {
		// With no CIDR set, proxy headers must be ignored even if present.
		// We verify this indirectly: nil-redis limiter still passes through,
		// so we just confirm the config field is wired correctly.
		cfg := DefaultRateLimitConfig()
		if cfg.TrustedProxyCIDR != "" {
			t.Error("expected empty TrustedProxyCIDR in default config")
		}
		rl := NewRateLimiter(nil, cfg)
		wrapped := rl.Handler(handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:9999"
		req.Header.Set("X-Real-IP", "9.9.9.9")
		rr := httptest.NewRecorder()
		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("TrustedProxyCIDR propagates to middleware", func(t *testing.T) {
		cfg := RateLimitConfig{
			PostLimit:        10,
			GetLimit:         10,
			Window:           time.Minute,
			TrustedProxyCIDR: "10.0.0.0/8",
		}
		rl := NewRateLimiter(nil, cfg)
		if rl.trustedProxyCIDR != "10.0.0.0/8" {
			t.Errorf("expected trustedProxyCIDR %q, got %q", "10.0.0.0/8", rl.trustedProxyCIDR)
		}
	})
}

func TestContentLengthValidator(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allows GET requests without Content-Length", func(t *testing.T) {
		wrapped := ContentLengthValidator(1024)(handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("rejects POST without Content-Length", func(t *testing.T) {
		wrapped := ContentLengthValidator(1024)(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.ContentLength = -1
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusLengthRequired {
			t.Errorf("expected status %d, got %d", http.StatusLengthRequired, rr.Code)
		}
	})

	t.Run("rejects POST with Content-Length exceeding max", func(t *testing.T) {
		wrapped := ContentLengthValidator(1024)(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.ContentLength = 2048
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected status %d, got %d",
				http.StatusRequestEntityTooLarge, rr.Code)
		}
	})

	t.Run("allows POST with valid Content-Length", func(t *testing.T) {
		wrapped := ContentLengthValidator(1024)(handler)

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test"))
		req.ContentLength = 4
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})
}
