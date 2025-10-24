package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func testRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", handleHealth)
	r.Post("/create", handleCreate)
	r.Post("/read/{id}", handleRead)
	return r
}

func mustStartMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	return mr
}

func setTestRedis(t *testing.T, addr string) {
	t.Helper()
	rdb = redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
}

func httpJSON(t *testing.T, ts *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode json: %v", err)
		}
	}
	req, _ := http.NewRequest(method, ts.URL+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http do: %v", err)
	}
	return resp
}

func decodeJSON[T any](t *testing.T, resp *http.Response, out *T) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func TestCreateAndReadSuccess_DeleteOnRead(t *testing.T) {
	mr := mustStartMiniRedis(t)
	defer mr.Close()
	setTestRedis(t, mr.Addr())

	ts := httptest.NewServer(testRouter())
	defer ts.Close()

	type createResp struct {
		ID        string    `json:"id"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	crReq := map[string]string{
		"secret":     "top secret",
		"passphrase": "swordfish3",
		"expiry":     "1h",
	}
	resp := httpJSON(t, ts, "POST", "/create", crReq)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var cr createResp
	decodeJSON(t, resp, &cr)
	if cr.ID == "" {
		t.Fatalf("empty id")
	}

	// key exists in Redis and has TTL
	key := redisKey(cr.ID)
	if !mr.Exists(key) {
		t.Fatalf("secret not stored in redis")
	}
	ttl := mr.TTL(key)
	if ttl <= 0 || ttl > time.Hour+time.Minute {
		t.Fatalf("unexpected TTL: %v", ttl)
	}

	rdReq := map[string]string{"passphrase": "swordfish3"}
	resp = httpJSON(t, ts, "POST", "/read/"+cr.ID, rdReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var rd map[string]string
	decodeJSON(t, resp, &rd)
	if rd["secret"] != "top secret" {
		t.Fatalf("decrypted secret mismatch: %q", rd["secret"])
	}

	// after successful read: key must be deleted
	if mr.Exists(key) {
		t.Fatalf("secret key still exists after delete-on-read")
	}

	// second read should be 404
	resp = httpJSON(t, ts, "POST", "/read/"+cr.ID, rdReq)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete-on-read, got %d", resp.StatusCode)
	}
}

func TestWrongPassphraseThreeTimesDeletesSecret(t *testing.T) {
	mr := mustStartMiniRedis(t)
	defer mr.Close()
	setTestRedis(t, mr.Addr())

	ts := httptest.NewServer(testRouter())
	defer ts.Close()

	crReq := map[string]string{
		"secret":     "nuclear codes",
		"passphrase": "correct-horse3",
		"expiry":     "6h",
	}
	resp := httpJSON(t, ts, "POST", "/create", crReq)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var cr struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &cr)
	key := redisKey(cr.ID)
	attKey := attemptsKey(cr.ID)

	// 1st wrong attempt
	resp = httpJSON(t, ts, "POST", "/read/"+cr.ID, map[string]string{"passphrase": "nope"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 (1st wrong), got %d", resp.StatusCode)
	}
	if !mr.Exists(key) {
		t.Fatalf("secret should remain after 1st wrong attempt")
	}
	if got, err := mr.Get(attKey); err != nil || got != "1" {
		t.Fatalf("expected attempts=1, got %q err=%v", got, err)
	}

	// 2nd wrong attempt
	resp = httpJSON(t, ts, "POST", "/read/"+cr.ID, map[string]string{"passphrase": "still-wrong"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 (2nd wrong), got %d", resp.StatusCode)
	}
	if got, err := mr.Get(attKey); err != nil || got != "2" {
		t.Fatalf("expected attempts=2, got %q err=%v", got, err)
	}
	if !mr.Exists(key) {
		t.Fatalf("secret should remain after 2nd wrong attempt")
	}

	// 3rd wrong attempt, secret deleted
	resp = httpJSON(t, ts, "POST", "/read/"+cr.ID, map[string]string{"passphrase": "third-time-wrong"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 (3rd wrong), got %d", resp.StatusCode)
	}
	if mr.Exists(key) {
		t.Fatalf("secret should be deleted after 3rd wrong attempt")
	}
	// attempts key should also be gone (script deletes both)
	if mr.Exists(attKey) {
		t.Fatalf("attempts key should be deleted after reaching threshold")
	}

	// now any read returns 404
	resp = httpJSON(t, ts, "POST", "/read/"+cr.ID, map[string]string{"passphrase": "correct-horse3"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after secret deleted, got %d", resp.StatusCode)
	}
}

func TestInvalidExpiryAndMissingFields(t *testing.T) {
	mr := mustStartMiniRedis(t)
	defer mr.Close()
	setTestRedis(t, mr.Addr())
	ts := httptest.NewServer(testRouter())
	defer ts.Close()

	// invalid expiry
	resp := httpJSON(t, ts, "POST", "/create", map[string]string{
		"secret":     "x",
		"passphrase": "portering3",
		"expiry":     "42h",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid expiry, got %d", resp.StatusCode)
	}

	// missing passphrase
	resp = httpJSON(t, ts, "POST", "/create", map[string]string{
		"secret": "x",
		"expiry": "1h",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing passphrase, got %d", resp.StatusCode)
	}

	// missing secret
	resp = httpJSON(t, ts, "POST", "/create", map[string]string{
		"passphrase": "portering3",
		"expiry":     "1h",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing secret, got %d", resp.StatusCode)
	}
}

func TestReadNotFound(t *testing.T) {
	mr := mustStartMiniRedis(t)
	defer mr.Close()
	setTestRedis(t, mr.Addr())
	ts := httptest.NewServer(testRouter())
	defer ts.Close()

	resp := httpJSON(t, ts, "POST", "/read/00000000-0000-0000-0000-000000000000", map[string]string{"passphrase": "anything3"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing id, got %d", resp.StatusCode)
	}
}

func TestHealth(t *testing.T) {
	ts := httptest.NewServer(testRouter())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}
