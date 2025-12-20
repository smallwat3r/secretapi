package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/smallwat3r/secretapi/internal/domain"
)

func TestCreateSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/create" {
			t.Errorf("Expected to request '/create', got: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' method, got: %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		// Use a map to simulate the exact JSON response from the server
		response := map[string]interface{}{
			"id":         "test-id",
			"passcode":   "test-passcode",
			"expires_at": time.Now().Add(24 * time.Hour),
			"read_url":   "http://localhost/read/test-id",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	createSecret(server.URL, "test-secret", "")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = oldStdout

	if !strings.Contains(buf.String(), "URL:") {
		t.Errorf("Expected output to contain 'URL:', got '%s'", buf.String())
	}
	if !strings.Contains(buf.String(), "Passcode:") {
		t.Errorf("Expected output to contain 'Passcode:', got '%s'", buf.String())
	}
	if !strings.Contains(buf.String(), "Expires:") {
		t.Errorf("Expected output to contain 'Expires:', got '%s'", buf.String())
	}
}

func TestReadSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/read/") {
			t.Errorf("Expected to request '/read/:key', got: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' method, got: %s", r.Method)
		}
		if r.Header.Get("X-Passcode") != "test-passcode" {
			t.Errorf("Expected 'X-Passcode' header to be 'test-passcode', got: %s", r.Header.Get("X-Passcode"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected 'Accept' header to be 'application/json', got: %s", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(domain.ReadRes{
			Secret: "test-secret",
		})
	}))
	defer server.Close()

	testCases := []struct {
		name string
		url  string
	}{
		{"without trailing slash", server.URL + "/read/test-id"},
		{"with trailing slash", server.URL + "/read/test-id/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			readSecret(tc.url, "test-passcode")

			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			os.Stdout = oldStdout

			expected := "test-secret\n"
			if buf.String() != expected {
				t.Errorf("Expected output to be '%s', got '%s'", expected, buf.String())
			}
		})
	}
}

func TestPrintUsage(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printUsage()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = oldStdout

	if !strings.Contains(buf.String(), "Usage:") {
		t.Errorf("Expected output to contain 'Usage:', got '%s'", buf.String())
	}
	if !strings.Contains(buf.String(), "create") {
		t.Errorf("Expected output to contain 'create', got '%s'", buf.String())
	}
	if !strings.Contains(buf.String(), "read") {
		t.Errorf("Expected output to contain 'read', got '%s'", buf.String())
	}
	if !strings.Contains(buf.String(), "help") {
		t.Errorf("Expected output to contain 'help', got '%s'", buf.String())
	}
}
