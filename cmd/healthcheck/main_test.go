package main

import (
	"net"
	"net/http"
	"testing"
)

func testServer(
	t *testing.T, status int,
) string {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(
		w http.ResponseWriter, r *http.Request,
	) {
		w.WriteHeader(status)
	})

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { srv.Close() })

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	return port
}

func TestCheck(t *testing.T) {
	t.Run("returns nil when server is healthy", func(t *testing.T) {
		port := testServer(t, http.StatusOK)
		if err := check(port); err != nil {
			t.Fatalf("expected healthy, got error: %v", err)
		}
	})

	t.Run("returns error on unhealthy status", func(t *testing.T) {
		port := testServer(t, http.StatusServiceUnavailable)
		if err := check(port); err == nil {
			t.Fatal("expected error for unhealthy status")
		}
	})

	t.Run("returns error when no server is running", func(t *testing.T) {
		if err := check("0"); err == nil {
			t.Fatal("expected error when no server running")
		}
	})
}
