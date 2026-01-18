package app

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/smallwat3r/secretapi/internal/utility"

	"github.com/redis/go-redis/v9"
)

// ContentLengthValidator validates Content-Length header for requests with bodies.
// It rejects requests without Content-Length or with excessive Content-Length.
func ContentLengthValidator(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate methods that typically have request bodies
			if r.Method == http.MethodPost || r.Method == http.MethodPut ||
				r.Method == http.MethodPatch {
				// Check if Content-Length header is present
				// r.ContentLength is -1 if not specified or chunked encoding
				if r.ContentLength < 0 {
					utility.HttpError(w, http.StatusLengthRequired,
						"Content-Length header is required")
					return
				}
				// Reject if Content-Length exceeds maximum
				if r.ContentLength > maxSize {
					utility.HttpError(w, http.StatusRequestEntityTooLarge,
						"Content-Length exceeds maximum allowed size")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersConfig holds configuration for security headers middleware.
type SecurityHeadersConfig struct {
	RequireHTTPS bool
}

// SecurityHeaders adds security-related HTTP headers to responses.
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// HTTPS enforcement with HSTS
			// Skip redirect for /health endpoint to allow internal health checks
			if cfg.RequireHTTPS && r.URL.Path != "/health" {
				// Check if request is over HTTPS (direct TLS or via proxy)
				isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
				if !isHTTPS {
					// Redirect HTTP to HTTPS
					target := "https://" + r.Host + r.URL.RequestURI()
					http.Redirect(w, r, target, http.StatusMovedPermanently)
					return
				}
				// HSTS: instruct browsers to only use HTTPS for 1 year
				w.Header().Set("Strict-Transport-Security",
					"max-age=31536000; includeSubDomains")
			}

			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// Prevent clickjacking (also enforced by CSP frame-ancestors)
			w.Header().Set("X-Frame-Options", "DENY")
			// Control referrer information
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			// Content Security Policy
			csp := "default-src 'self'; script-src 'self'; " +
				"style-src 'self' 'unsafe-inline'; " +
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
		PostLimit: 30,  // 30 POST requests per minute
		GetLimit:  120, // 120 GET requests per minute
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

		// Use a pipeline to atomically increment and set expiry.
		// This avoids a race condition where the process could crash between
		// INCR and EXPIRE, leaving a key without TTL.
		pipe := m.rdb.TxPipeline()
		incr := pipe.Incr(r.Context(), key)
		pipe.Expire(r.Context(), key, m.window)
		_, err := pipe.Exec(r.Context())
		if err != nil {
			log.Printf("rate limit redis error: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		if int(incr.Val()) > limit {
			utility.HttpError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}
