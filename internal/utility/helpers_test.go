package utility

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestParseExpiry(t *testing.T) {
	testCases := []struct {
		input       string
		expected    time.Duration
		shouldMatch bool
	}{
		{"1h", time.Hour, true},
		{"6h", 6 * time.Hour, true},
		{"1d", 24 * time.Hour, true},
		{"3d", 72 * time.Hour, true},
		{"", 0, false},
		{"1m", 0, false},
		{"1y", 0, false},
		{"24h", 0, false},
		{"invalid", 0, false},
		{"1H", 0, false}, // case sensitive
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			duration, ok := ParseExpiry(tc.input)
			if ok != tc.shouldMatch {
				t.Errorf("ParseExpiry(%q) ok=%v, want %v",
					tc.input, ok, tc.shouldMatch)
			}
			if tc.shouldMatch && duration != tc.expected {
				t.Errorf("ParseExpiry(%q) = %v, want %v",
					tc.input, duration, tc.expected)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	t.Run("sets correct headers and status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteJSON(rr, http.StatusOK, map[string]string{"key": "value"})

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
	})

	t.Run("encodes struct correctly", func(t *testing.T) {
		rr := httptest.NewRecorder()

		type testStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		WriteJSON(rr, http.StatusCreated, testStruct{Name: "test", Value: 42})

		expected := `{"name":"test","value":42}`
		// Response includes newline from json.Encoder
		got := rr.Body.String()
		if got != expected+"\n" {
			t.Errorf("expected body %q, got %q", expected+"\n", got)
		}
	})

	t.Run("handles nil value", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteJSON(rr, http.StatusOK, nil)

		if rr.Body.String() != "null\n" {
			t.Errorf("expected null, got %q", rr.Body.String())
		}
	})
}

func TestHttpError(t *testing.T) {
	rr := httptest.NewRecorder()
	HttpError(rr, http.StatusBadRequest, "something went wrong")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	expected := `{"error":"something went wrong"}`
	got := rr.Body.String()
	if got != expected+"\n" {
		t.Errorf("expected body %q, got %q", expected+"\n", got)
	}
}

func TestGetenv(t *testing.T) {
	t.Run("returns environment variable when set", func(t *testing.T) {
		key := "TEST_GETENV_VAR"
		expected := "test_value"
		os.Setenv(key, expected)
		defer os.Unsetenv(key)

		got := Getenv(key, "default")
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		key := "TEST_GETENV_UNSET_VAR"
		os.Unsetenv(key)

		expected := "default_value"
		got := Getenv(key, expected)
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("returns default when empty", func(t *testing.T) {
		key := "TEST_GETENV_EMPTY_VAR"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		expected := "default_value"
		got := Getenv(key, expected)
		if got != expected {
			t.Errorf("expected %q for empty var, got %q", expected, got)
		}
	})
}

func TestIntPtr(t *testing.T) {
	testCases := []int{0, 1, -1, 42, 1000000}

	for _, val := range testCases {
		ptr := IntPtr(val)
		if ptr == nil {
			t.Errorf("IntPtr(%d) returned nil", val)
			continue
		}
		if *ptr != val {
			t.Errorf("IntPtr(%d) = %d, want %d", val, *ptr, val)
		}
	}
}
