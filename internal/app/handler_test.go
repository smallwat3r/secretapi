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

	"secretapi/internal/domain"
	"secretapi/pkg/utility"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type mockSecretRepository struct {
	StoreSecretFunc            func(id string, secret []byte, ttl time.Duration) error
	GetSecretFunc              func(id string) ([]byte, error)
	DelIfMatchFunc             func(id string, old []byte)
	IncrFailAndMaybeDeleteFunc func(id string)
	DeleteAttemptsFunc         func(id string) error
}

func (m *mockSecretRepository) StoreSecret(id string, secret []byte, ttl time.Duration) error {
	if m.StoreSecretFunc != nil {
		return m.StoreSecretFunc(id, secret, ttl)
	}
	return nil
}

func (m *mockSecretRepository) GetSecret(id string) ([]byte, error) {
	if m.GetSecretFunc != nil {
		return m.GetSecretFunc(id)
	}
	return nil, nil
}

func (m *mockSecretRepository) DelIfMatch(id string, old []byte) {
	if m.DelIfMatchFunc != nil {
		m.DelIfMatchFunc(id, old)
	}
}

func (m *mockSecretRepository) IncrFailAndMaybeDelete(id string) {
	if m.IncrFailAndMaybeDeleteFunc != nil {
		m.IncrFailAndMaybeDeleteFunc(id)
	}
}

func (m *mockSecretRepository) DeleteAttempts(id string) error {
	if m.DeleteAttemptsFunc != nil {
		return m.DeleteAttemptsFunc(id)
	}
	return nil
}

func lowerCryptoParamsForTest(t *testing.T) {
	t.Helper()
	originalArgonTime := utility.ArgonTime
	originalArgonMemory := utility.ArgonMemory
	utility.ArgonTime = 1
	utility.ArgonMemory = 1024
	t.Cleanup(func() {
		utility.ArgonTime = originalArgonTime
		utility.ArgonMemory = originalArgonMemory
	})
}

func TestHandler_HandleHealth(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/health/", nil)
	rr := httptest.NewRecorder()

	handler.HandleHealth(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if body := rr.Body.String(); body != "ok" {
		t.Errorf("handler returned unexpected body: got %v want %v", body, "ok")
	}
}

func TestHandler_HandleCreate(t *testing.T) {
	lowerCryptoParamsForTest(t)

	mockRepo := &mockSecretRepository{}
	handler := NewHandler(mockRepo)

	t.Run("successful creation", func(t *testing.T) {
		mockRepo.StoreSecretFunc = func(id string, secret []byte, ttl time.Duration) error {
			return nil
		}
		reqBody := `{"secret":"my-secret","expiry":"1h"}`
		req := httptest.NewRequest(http.MethodPost, "/create/", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()

		handler.HandleCreate(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
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
		if !strings.HasSuffix(res.ReadURL, "?format=plain") {
			t.Error("expected URL to contain format=plain")
		}
	})

	t.Run("successful creation with default expiry", func(t *testing.T) {
		var capturedTTL time.Duration
		mockRepo.StoreSecretFunc = func(id string, secret []byte, ttl time.Duration) error {
			capturedTTL = ttl
			return nil
		}
		reqBody := `{"secret":"my-secret"}`
		req := httptest.NewRequest(http.MethodPost, "/create/", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()

		handler.HandleCreate(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
		if capturedTTL != 24*time.Hour {
			t.Errorf("expected ttl to be 24h, got %v", capturedTTL)
		}
	})

	t.Run("bad request - invalid json", func(t *testing.T) {
		reqBody := `{"secret":`
		req := httptest.NewRequest(http.MethodPost, "/create/", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("bad request - missing secret", func(t *testing.T) {
		reqBody := `{}`
		req := httptest.NewRequest(http.MethodPost, "/create/", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("bad request - invalid expiry", func(t *testing.T) {
		reqBody := `{"secret":"my-secret","expiry":"1y"}`
		req := httptest.NewRequest(http.MethodPost, "/create/", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("internal server error - store secret fails", func(t *testing.T) {
		mockRepo.StoreSecretFunc = func(id string, secret []byte, ttl time.Duration) error {
			return errors.New("db error")
		}
		reqBody := `{"secret":"my-secret","expiry":"1h"}`
		req := httptest.NewRequest(http.MethodPost, "/create/", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()
		handler.HandleCreate(rr, req)
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})
}

func TestHandler_HandleRead(t *testing.T) {
	lowerCryptoParamsForTest(t)

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
		mockRepo.GetSecretFunc = func(id string) ([]byte, error) {
			if id == secretID {
				return encryptedSecret, nil
			}
			return nil, redis.Nil
		}
		mockRepo.DelIfMatchFunc = func(id string, old []byte) {}
		mockRepo.DeleteAttemptsFunc = func(id string) error { return nil }

		req := httptest.NewRequest(http.MethodPost, "/read/"+secretID+"/", nil)
		req.Header.Set("X-Passcode", passcode)

		// add chi URL param context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", secretID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		var res domain.ReadRes
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if res.Secret != secretText {
			t.Errorf("handler returned wrong secret: got %v want %v", res.Secret, secretText)
		}
	})

	t.Run("successful read in plain format", func(t *testing.T) {
		mockRepo.GetSecretFunc = func(id string) ([]byte, error) {
			if id == secretID {
				return encryptedSecret, nil
			}
			return nil, redis.Nil
		}
		mockRepo.DelIfMatchFunc = func(id string, old []byte) {}
		mockRepo.DeleteAttemptsFunc = func(id string) error { return nil }

		target := &url.URL{
			Path:     "/read/" + secretID + "/",
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
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if body := rr.Body.String(); body != secretText {
			t.Errorf("handler returned wrong secret: got %v want %v", body, secretText)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "text/plain" {
			t.Errorf("handler returned wrong content type: got %v want %v", contentType, "text/plain")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo.GetSecretFunc = func(id string) ([]byte, error) {
			return nil, redis.Nil
		}
		req := httptest.NewRequest(http.MethodPost, "/read/wrong-id/", nil)
		req.Header.Set("X-Passcode", passcode)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "wrong-id")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("unauthorized - wrong passcode", func(t *testing.T) {
		var incrCalled bool
		mockRepo.GetSecretFunc = func(id string) ([]byte, error) {
			return encryptedSecret, nil
		}
		mockRepo.IncrFailAndMaybeDeleteFunc = func(id string) {
			incrCalled = true
		}
		req := httptest.NewRequest(http.MethodPost, "/read/"+secretID+"/", nil)
		req.Header.Set("X-Passcode", "wrong-pass")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", secretID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		handler.HandleRead(rr, req)
		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !incrCalled {
			t.Error("expected IncrFailAndMaybeDelete to be called")
		}
	})
}
