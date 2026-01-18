package app

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/smallwat3r/secretapi/internal/domain"
	"github.com/smallwat3r/secretapi/internal/utility"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type Handler struct {
	repo domain.SecretRepository
}

func NewHandler(repo domain.SecretRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxRequestBodySize)

	var req domain.CreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			utility.HttpError(w, http.StatusRequestEntityTooLarge,
				"request body too large")
			return
		}
		utility.HttpError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Secret = strings.TrimSpace(req.Secret)
	if req.Secret == "" {
		utility.HttpError(w, http.StatusBadRequest, "secret is required")
		return
	}
	if len(req.Secret) > domain.MaxSecretSize {
		utility.HttpError(w, http.StatusRequestEntityTooLarge, "secret exceeds 64KB limit")
		return
	}

	passcode, err := utility.GeneratePasscode()
	if err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "passcode generation failed")
		return
	}

	var ttl time.Duration
	if req.Expiry == "" {
		ttl = domain.DefaultExpiry
	} else {
		var ok bool
		ttl, ok = utility.ParseExpiry(req.Expiry)
		if !ok {
			utility.HttpError(w, http.StatusBadRequest,
				"expiry must be one of: 1h, 6h, 1d, 3d")
			return
		}
	}

	blob, err := utility.Encrypt([]byte(req.Secret), passcode)
	if err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "encryption failed")
		return
	}

	id := uuid.NewString()

	if err := h.repo.StoreSecret(r.Context(), id, blob, ttl); err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "failed to store secret")
		return
	}

	log.Printf("secret created: id=%s expiry=%s", id, ttl)

	expiresAt := time.Now().Add(ttl).UTC()

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	readURL := &url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   "/read/" + id,
	}

	utility.WriteJSON(
		w,
		http.StatusCreated,
		domain.CreateRes{
			ID:        id,
			Passcode:  passcode,
			ExpiresAt: expiresAt,
			ReadURL:   readURL.String(),
		},
	)
}

func (h *Handler) HandleRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utility.HttpError(w, http.StatusBadRequest, "missing id")
		return
	}
	passcode := r.Header.Get("X-Passcode")
	if passcode == "" {
		utility.HttpError(w, http.StatusBadRequest, "passcode is required")
		return
	}

	blob, err := h.repo.GetSecret(r.Context(), id)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			utility.HttpError(w, http.StatusNotFound, "not found or expired")
			return
		}
		utility.HttpError(w, http.StatusInternalServerError, "failed to fetch secret")
		return
	}

	plaintext, err := utility.Decrypt(blob, passcode)
	if err != nil {
		// wrong passcode
		log.Printf("invalid passcode for secret: id=%s", id)
		attempts, _ := h.repo.IncrFailAndMaybeDelete(r.Context(), id)
		utility.WriteJSON(w, http.StatusUnauthorized, domain.ReadRes{
			RemainingAttempts: utility.IntPtr(domain.MaxReadAttempts - int(attempts)),
		})
		return
	}

	// successful decrypt
	log.Printf("secret successfully read: id=%s", id)
	ctx := r.Context()
	_ = h.repo.DelIfMatch(ctx, id, blob)
	// tidy up attempts counter in background with detached context
	go func() {
		if err := h.repo.DeleteAttempts(context.Background(), id); err != nil {
			log.Printf("failed to delete attempts counter: id=%s err=%v", id, err)
		}
	}()

	format := r.URL.Query().Get("format")
	if format == "plain" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(plaintext)
		return
	}

	utility.WriteJSON(w, http.StatusOK, domain.ReadRes{Secret: string(plaintext)})
}

func (h *Handler) HandleIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, "web/static/dist/index.html")
}

func (h *Handler) HandleRobotsTXT(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/robots.txt")
}

// SecurityHeaders adds security-related HTTP headers to responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Prevent clickjacking (also enforced by CSP frame-ancestors)
		w.Header().Set("X-Frame-Options", "DENY")
		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Content Security Policy
		csp := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data:; font-src 'self'; connect-src 'self'; " +
			"frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
		w.Header().Set("Content-Security-Policy", csp)
		// Restrict browser features
		w.Header().Set("Permissions-Policy",
			"geolocation=(), microphone=(), camera=(), payment=(), usb=()")
		// Isolate browsing context
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

		next.ServeHTTP(w, r)
	})
}

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	PostRate     rate.Limit    // requests per second for POST
	PostBurst    int           // burst size for POST
	GetRate      rate.Limit    // requests per second for GET
	GetBurst     int           // burst size for GET
	CleanupEvery time.Duration // how often to clean up stale entries
	EntryTTL     time.Duration // how long to keep an entry after last use
}

// DefaultRateLimitConfig returns sensible default rate limits.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		PostRate:     2,
		PostBurst:    5,
		GetRate:      10,
		GetBurst:     20,
		CleanupEvery: 5 * time.Minute,
		EntryTTL:     10 * time.Minute,
	}
}

// ipRateLimiter tracks rate limiters per IP address.
type ipRateLimiter struct {
	mu        sync.RWMutex
	limiters  map[string]*rateLimiterEntry
	postRate  rate.Limit
	postBurst int
	getRate   rate.Limit
	getBurst  int
	entryTTL  time.Duration
	done      chan struct{}
}

type rateLimiterEntry struct {
	post     *rate.Limiter
	get      *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(cfg RateLimitConfig) *ipRateLimiter {
	rl := &ipRateLimiter{
		limiters:  make(map[string]*rateLimiterEntry),
		postRate:  cfg.PostRate,
		postBurst: cfg.PostBurst,
		getRate:   cfg.GetRate,
		getBurst:  cfg.GetBurst,
		entryTTL:  cfg.EntryTTL,
		done:      make(chan struct{}),
	}

	go rl.cleanupLoop(cfg.CleanupEvery)
	return rl
}

func (rl *ipRateLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.done:
			return
		}
	}
}

func (rl *ipRateLimiter) stop() {
	close(rl.done)
}

func (rl *ipRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, entry := range rl.limiters {
		if now.Sub(entry.lastSeen) > rl.entryTTL {
			delete(rl.limiters, ip)
		}
	}
}

func (rl *ipRateLimiter) getLimiter(ip string) *rateLimiterEntry {
	rl.mu.RLock()
	entry, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if exists {
		rl.mu.Lock()
		entry.lastSeen = time.Now()
		rl.mu.Unlock()
		return entry
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// double-check after acquiring write lock
	if entry, exists = rl.limiters[ip]; exists {
		entry.lastSeen = time.Now()
		return entry
	}

	entry = &rateLimiterEntry{
		post:     rate.NewLimiter(rl.postRate, rl.postBurst),
		get:      rate.NewLimiter(rl.getRate, rl.getBurst),
		lastSeen: time.Now(),
	}
	rl.limiters[ip] = entry
	return entry
}

// RateLimiterMiddleware wraps rate limiting logic with cleanup capability.
type RateLimiterMiddleware struct {
	rl *ipRateLimiter
}

// NewRateLimiter creates a new rate limiter middleware.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		rl: newIPRateLimiter(cfg),
	}
}

// Handler returns the HTTP middleware handler.
func (m *RateLimiterMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		// chi's RealIP middleware sets this
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			ip = realIP
		} else if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			// take the first IP from the list
			if idx := strings.Index(forwardedFor, ","); idx != -1 {
				ip = strings.TrimSpace(forwardedFor[:idx])
			} else {
				ip = strings.TrimSpace(forwardedFor)
			}
		}

		entry := m.rl.getLimiter(ip)

		var limiter *rate.Limiter
		switch r.Method {
		case http.MethodPost:
			limiter = entry.post
		case http.MethodGet:
			limiter = entry.get
		}

		if limiter != nil && !limiter.Allow() {
			utility.HttpError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Stop terminates the cleanup goroutine.
func (m *RateLimiterMiddleware) Stop() {
	m.rl.stop()
}
