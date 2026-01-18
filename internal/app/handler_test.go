package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/smallwat3r/secretapi/internal/domain"
	"github.com/smallwat3r/secretapi/internal/utility"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type mockSecretRepository struct {
	StoreSecretFunc func(ctx context.Context, id string, secret []byte,
		ttl time.Duration) error
	GetSecretFunc              func(ctx context.Context, id string) ([]byte, error)
	DelIfMatchFunc             func(ctx context.Context, id string, old []byte) error
	IncrFailAndMaybeDeleteFunc func(ctx context.Context, id string) (int64, error)
	DeleteAttemptsFunc         func(ctx context.Context, id string) error
}

func (m *mockSecretRepository) StoreSecret(
	ctx context.Context, id string, secret []byte, ttl time.Duration,
) error {
	if m.StoreSecretFunc != nil {
		return m.StoreSecretFunc(ctx, id, secret, ttl)
	}
	return nil
}

func (m *mockSecretRepository) GetSecret(ctx context.Context, id string) ([]byte, error) {
	if m.GetSecretFunc != nil {
		return m.GetSecretFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockSecretRepository) DelIfMatch(ctx context.Context, id string, old []byte) error {
	if m.DelIfMatchFunc != nil {
		return m.DelIfMatchFunc(ctx, id, old)
	}
	return nil
}

func (m *mockSecretRepository) IncrFailAndMaybeDelete(
	ctx context.Context, id string,
) (int64, error) {
	if m.IncrFailAndMaybeDeleteFunc != nil {
		return m.IncrFailAndMaybeDeleteFunc(ctx, id)
	}
	return 0, nil
}

func (m *mockSecretRepository) DeleteAttempts(ctx context.Context, id string) error {
	if m.DeleteAttemptsFunc != nil {
		return m.DeleteAttemptsFunc(ctx, id)
	}
	return nil
}

func TestHandler_HandleHealth(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.HandleHealth(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", status, http.StatusOK)
	}
	if body := rr.Body.String(); body != "ok" {
		t.Errorf("handler returned unexpected body: got %v want %v", body, "ok")
	}
}

func TestHandler_HandleConfig(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rr := httptest.NewRecorder()

	handler.HandleConfig(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", status, http.StatusOK)
	}

	var res domain.ConfigRes
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if res.MaxSecretSize != domain.MaxSecretSize {
		t.Errorf("wrong max_secret_size: got %v want %v",
			res.MaxSecretSize, domain.MaxSecretSize)
	}
	if len(res.ExpiryOptions) == 0 {
		t.Error("expected expiry_options to be non-empty")
	}
}

func TestHandler_HandleCreate(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{}
	handler := NewHandler(mockRepo)

	t.Run("successful creation", func(t *testing.T) {
		mockRepo.StoreSecretFunc = func(
			ctx context.Context, id string, secret []byte, ttl time.Duration,
		) error {
			return nil
		}
		reqBody := `{"secret":"my-secret","expiry":"1h"}`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()

		handler.HandleCreate(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("wrong status code: got %v want %v", status, http.StatusCreated)
		}
		var res domain.CreateRes
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if res.ID == "" {
			t.Error("expected non-empty ID in response")
		}
		if res.Passcode == "" {
			t.Error("expected non-empty passcode in response")
		}
		if res.ReadURL == "" {
			t.Error("expected non-empty URL in response")
		}
		if !strings.Contains(res.ReadURL, res.ID) {
			t.Error("expected URL to contain the secret ID")
		}
	})

	t.Run("successful creation with default expiry", func(t *testing.T) {
		var capturedTTL time.Duration
		mockRepo.StoreSecretFunc = func(
			ctx context.Context, id string, secret []byte, ttl time.Duration,
		) error {
			capturedTTL = ttl
			return nil
		}
		reqBody := `{"secret":"my-secret"}`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()

		handler.HandleCreate(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("wrong status code: got %v want %v", status, http.StatusCreated)
		}
		if capturedTTL != 24*time.Hour {
			t.Errorf("expected ttl to be 24h, got %v", capturedTTL)
		}
	})

	t.Run("bad request - invalid json", func(t *testing.T) {
		reqBody := `{"secret":`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("wrong status: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("bad request - missing secret", func(t *testing.T) {
		reqBody := `{}`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("wrong status: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("bad request - invalid expiry", func(t *testing.T) {
		reqBody := `{"secret":"my-secret","expiry":"1y"}`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("wrong status: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("bad request - secret too large", func(t *testing.T) {
		largeSecret := strings.Repeat("a", 64*1024+1)
		reqBody := `{"secret":"` + largeSecret + `"}`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusRequestEntityTooLarge {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusRequestEntityTooLarge)
		}
	})

	t.Run("internal server error - store secret fails", func(t *testing.T) {
		mockRepo.StoreSecretFunc = func(
			ctx context.Context, id string, secret []byte, ttl time.Duration,
		) error {
			return errors.New("db error")
		}
		reqBody := `{"secret":"my-secret","expiry":"1h"}`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusInternalServerError)
		}
	})
}

func TestHandler_HandleRead(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{}
	handler := NewHandler(mockRepo)
	secretID := "test-id"
	passcode, err := utility.GeneratePasscode()
	if err != nil {
		t.Fatalf("failed to generate passcode: %v", err)
	}
	secretText := "my-secret"
	encryptedSecret, _ := utility.Encrypt([]byte(secretText), passcode)

	t.Run("successful read", func(t *testing.T) {
		mockRepo.GetSecretFunc = func(ctx context.Context, id string) ([]byte, error) {
			if id == secretID {
				return encryptedSecret, nil
			}
			return nil, redis.Nil
		}
		mockRepo.DelIfMatchFunc = func(
			ctx context.Context, id string, old []byte,
		) error {
			return nil
		}
		mockRepo.DeleteAttemptsFunc = func(ctx context.Context, id string) error {
			return nil
		}

		req := httptest.NewRequest(http.MethodPost, "/read/"+secretID, nil)
		req.Header.Set("X-Passcode", passcode)

		// add chi URL param context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", secretID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("wrong status code: got %v want %v", status, http.StatusOK)
		}
		var res domain.ReadRes
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if res.Secret != secretText {
			t.Errorf("wrong secret: got %v want %v", res.Secret, secretText)
		}
	})

	t.Run("successful read in plain format", func(t *testing.T) {
		mockRepo.GetSecretFunc = func(ctx context.Context, id string) ([]byte, error) {
			if id == secretID {
				return encryptedSecret, nil
			}
			return nil, redis.Nil
		}
		mockRepo.DelIfMatchFunc = func(
			ctx context.Context, id string, old []byte,
		) error {
			return nil
		}
		mockRepo.DeleteAttemptsFunc = func(ctx context.Context, id string) error {
			return nil
		}

		target := &url.URL{
			Path:     "/read/" + secretID,
			RawQuery: "format=plain",
		}
		req := httptest.NewRequest(http.MethodPost, target.String(), nil)
		req.Header.Set("X-Passcode", passcode)

		// add chi URL param context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", secretID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("wrong status code: got %v want %v", status, http.StatusOK)
		}
		if body := rr.Body.String(); body != secretText {
			t.Errorf("handler returned wrong secret: got %v want %v", body, secretText)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "text/plain" {
			t.Errorf("wrong content type: got %v want %v",
				contentType, "text/plain")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo.GetSecretFunc = func(ctx context.Context, id string) ([]byte, error) {
			return nil, redis.Nil
		}
		req := httptest.NewRequest(http.MethodPost, "/read/wrong-id", nil)
		req.Header.Set("X-Passcode", passcode)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "wrong-id")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("wrong status: got %v want %v",
				status, http.StatusNotFound)
		}
	})

	t.Run("unauthorized - wrong passcode", func(t *testing.T) {
		mockRepo.GetSecretFunc = func(ctx context.Context, id string) ([]byte, error) {
			return encryptedSecret, nil
		}
		mockRepo.IncrFailAndMaybeDeleteFunc = func(
			ctx context.Context, id string,
		) (int64, error) {
			return 1, nil
		}
		req := httptest.NewRequest(http.MethodPost, "/read/"+secretID, nil)
		req.Header.Set("X-Passcode", "wrong-pass")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", secretID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)
		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("wrong status: got %v want %v", status, http.StatusUnauthorized)
		}

		var res domain.ReadRes
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if res.RemainingAttempts == nil {
			t.Fatal("expected remaining_attempts in response")
		}
		if *res.RemainingAttempts != 2 {
			t.Errorf("expected 2 remaining attempts, got %d", *res.RemainingAttempts)
		}
	})

	t.Run("bad request - missing passcode header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/read/"+secretID, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", secretID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("wrong status: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("bad request - missing id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/read/", nil)
		req.Header.Set("X-Passcode", passcode)
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("wrong status: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("internal server error - GetSecret fails", func(t *testing.T) {
		mockRepo.GetSecretFunc = func(ctx context.Context, id string) ([]byte, error) {
			return nil, errors.New("redis connection error")
		}
		req := httptest.NewRequest(http.MethodPost, "/read/"+secretID, nil)
		req.Header.Set("X-Passcode", passcode)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", secretID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusInternalServerError)
		}
	})
}

func TestHandler_HandleCreate_ExpiryOptions(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	testCases := []struct {
		expiry      string
		expectedTTL time.Duration
	}{
		{"1h", time.Hour},
		{"6h", 6 * time.Hour},
		{"1d", 24 * time.Hour},
		{"3d", 72 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run("expiry_"+tc.expiry, func(t *testing.T) {
			var capturedTTL time.Duration
			mockRepo := &mockSecretRepository{
				StoreSecretFunc: func(
					ctx context.Context, id string, secret []byte,
					ttl time.Duration,
				) error {
					capturedTTL = ttl
					return nil
				},
			}
			handler := NewHandler(mockRepo)

			reqBody := `{"secret":"test","expiry":"` + tc.expiry + `"}`
			req := httptest.NewRequest(
				http.MethodPost, "/create", strings.NewReader(reqBody))
			rr := httptest.NewRecorder()

			handler.HandleCreate(rr, req)

			if rr.Code != http.StatusCreated {
				t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
			}
			if capturedTTL != tc.expectedTTL {
				t.Errorf("expected TTL %v, got %v", tc.expectedTTL, capturedTTL)
			}
		})
	}
}

func TestHandler_HandleCreate_HTTPSDetection(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{
		StoreSecretFunc: func(
			ctx context.Context, id string, secret []byte, ttl time.Duration,
		) error {
			return nil
		},
	}
	handler := NewHandler(mockRepo)

	t.Run("detects HTTPS from X-Forwarded-Proto header", func(t *testing.T) {
		reqBody := `{"secret":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Host = "example.com"
		rr := httptest.NewRecorder()

		handler.HandleCreate(rr, req)

		var res domain.CreateRes
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if !strings.HasPrefix(res.ReadURL, "https://") {
			t.Errorf("expected HTTPS URL, got %s", res.ReadURL)
		}
	})

	t.Run("uses HTTP when no TLS indicators", func(t *testing.T) {
		reqBody := `{"secret":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
		req.Host = "example.com"
		rr := httptest.NewRecorder()

		handler.HandleCreate(rr, req)

		var res domain.CreateRes
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if !strings.HasPrefix(res.ReadURL, "http://") {
			t.Errorf("expected HTTP URL, got %s", res.ReadURL)
		}
	})
}

func TestHandler_HandleCreate_WhitespaceSecret(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{}
	handler := NewHandler(mockRepo)

	testCases := []struct {
		name   string
		secret string
	}{
		{"spaces only", "   "},
		{"tabs only", "\t\t"},
		{"newlines only", "\n\n"},
		{"mixed whitespace", "  \t\n  "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := `{"secret":"` + tc.secret + `"}`
			req := httptest.NewRequest(
				http.MethodPost, "/create", strings.NewReader(reqBody))
			rr := httptest.NewRecorder()

			handler.HandleCreate(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status %d for whitespace-only secret, got %d",
					http.StatusBadRequest, rr.Code)
			}
		})
	}
}

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

	// Verify X-XSS-Protection is NOT set (deprecated header)
	if got := rr.Header().Get("X-XSS-Protection"); got != "" {
		t.Errorf("X-XSS-Protection should not be set (deprecated), got %q", got)
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
}

func TestRateLimiter(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("passes through when redis is nil", func(t *testing.T) {
		rl := NewRateLimiter(nil, DefaultRateLimitConfig())
		wrapped := rl.Handler(handler)

		for i := 0; i < 100; i++ {
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
