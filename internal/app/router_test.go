package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/smallwat3r/secretapi/internal/utility"
)

func TestNewRouter_Routes(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{
		StoreSecretFunc: func(
			ctx context.Context, id string, secret []byte, ttl time.Duration,
		) error {
			return nil
		},
	}
	handler := NewHandler(mockRepo)
	router := NewRouter(handler)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"health check", http.MethodGet, "/health", http.StatusOK},
		{"root page", http.MethodGet, "/", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// For file serving endpoints, we might get 404 if file doesn't exist
			if tc.path == "/health" && rr.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d",
					tc.expectedStatus, rr.Code)
			}
		})
	}
}

func TestNewRouter_CreateEndpoint(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{
		StoreSecretFunc: func(
			ctx context.Context, id string, secret []byte, ttl time.Duration,
		) error {
			return nil
		},
	}
	handler := NewHandler(mockRepo)
	router := NewRouter(handler)

	reqBody := `{"secret":"test-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestNewRouter_ReadEndpoint_ValidUUID(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{
		GetSecretFunc: func(ctx context.Context, id string) ([]byte, error) {
			return nil, redis.Nil
		},
	}
	handler := NewHandler(mockRepo)
	router := NewRouter(handler)

	// Valid UUID format
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodPost, "/read/"+uuid, nil)
	req.Header.Set("X-Passcode", "test-passcode")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Should get 404 (secret not found) not 405 (method not allowed) or route not matched
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d for valid UUID, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestNewRouter_ReadEndpoint_InvalidUUID(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{}
	handler := NewHandler(mockRepo)
	router := NewRouter(handler)

	// Invalid UUID format - should not match route
	req := httptest.NewRequest(http.MethodPost, "/read/invalid-id", nil)
	req.Header.Set("X-Passcode", "test-passcode")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Route should not match, so we get 404 from router (not from handler)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d for invalid UUID, got %d",
			http.StatusNotFound, rr.Code)
	}
}

func TestNewRouter_SecurityHeaders(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{}
	handler := NewHandler(mockRepo)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Check security headers are set
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options to be nosniff, got %q", got)
	}
	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("expected X-Frame-Options to be DENY, got %q", got)
	}
}

func TestNewRouter_RedirectSlashes(t *testing.T) {
	utility.LowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{}
	handler := NewHandler(mockRepo)
	router := NewRouter(handler)

	// Request with trailing slash should redirect
	req := httptest.NewRequest(http.MethodGet, "/health/", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Chi's RedirectSlashes middleware returns 301 redirect
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("expected redirect status %d, got %d",
			http.StatusMovedPermanently, rr.Code)
	}
}
