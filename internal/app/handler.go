package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/smallwat3r/secretapi/internal/domain"
	"github.com/smallwat3r/secretapi/internal/utility"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

func (h *Handler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	utility.WriteJSON(w, http.StatusOK, domain.ConfigRes{
		MaxSecretSize: domain.MaxSecretSize,
		ExpiryOptions: domain.ExpiryOptions,
	})
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
	if err := h.repo.DelIfMatch(r.Context(), id, blob); err != nil {
		log.Printf("failed to delete secret after read: id=%s err=%v", id, err)
	}
	// tidy up attempts counter in background with timeout
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.repo.DeleteAttempts(ctx, id); err != nil {
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
	PostLimit int           // max POST requests per window
	GetLimit  int           // max GET requests per window
	Window    time.Duration // time window for rate limiting
}

// DefaultRateLimitConfig returns sensible default rate limits.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		PostLimit: 10,        // 10 POST requests per minute
		GetLimit:  60,        // 60 GET requests per minute
		Window:    time.Minute,
	}
}

// RateLimiterMiddleware uses Redis for distributed rate limiting.
type RateLimiterMiddleware struct {
	rdb       *redis.Client
	postLimit int
	getLimit  int
	window    time.Duration
}

// NewRateLimiter creates a new Redis-based rate limiter middleware.
func NewRateLimiter(rdb *redis.Client, cfg RateLimitConfig) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		rdb:       rdb,
		postLimit: cfg.PostLimit,
		getLimit:  cfg.GetLimit,
		window:    cfg.Window,
	}
}

// Handler returns the HTTP middleware handler.
func (m *RateLimiterMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting if Redis is not configured (e.g., in tests)
		if m.rdb == nil {
			next.ServeHTTP(w, r)
			return
		}

		ip := r.RemoteAddr
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			ip = realIP
		} else if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			if idx := strings.Index(forwardedFor, ","); idx != -1 {
				ip = strings.TrimSpace(forwardedFor[:idx])
			} else {
				ip = strings.TrimSpace(forwardedFor)
			}
		}

		var limit int
		switch r.Method {
		case http.MethodPost:
			limit = m.postLimit
		case http.MethodGet:
			limit = m.getLimit
		default:
			next.ServeHTTP(w, r)
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", ip, r.Method)
		count, err := m.rdb.Incr(r.Context(), key).Result()
		if err != nil {
			log.Printf("rate limit redis error: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			m.rdb.Expire(r.Context(), key, m.window)
		}

		if int(count) > limit {
			utility.HttpError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}
