package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/smallwat3r/secretapi/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

// cacheControl wraps an http.Handler to add Cache-Control headers.
func cacheControl(h http.Handler, maxAge time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control",
			fmt.Sprintf("public, max-age=%d", int(maxAge.Seconds())))
		h.ServeHTTP(w, r)
	})
}

func NewRouter(h *Handler, rdb *redis.Client, secCfg SecurityHeadersConfig) http.Handler {
	r := chi.NewRouter()
	rl := NewRateLimiter(rdb, DefaultRateLimitConfig())

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RedirectSlashes)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(SecurityHeaders(secCfg))
	r.Use(ContentLengthValidator(domain.MaxRequestBodySize))

	r.Get("/robots.txt", h.HandleRobotsTXT)
	r.Get("/health", h.HandleHealth)
	r.Get("/config", h.HandleConfig)

	fs := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*",
		http.StripPrefix("/static/", cacheControl(fs, 24*time.Hour)))

	// Page routes
	r.Get("/", h.HandleIndexHTML)
	r.Get("/about", h.HandleIndexHTML)
	r.Get("/read/{id:[0-9a-fA-F-]{36}}", h.HandleIndexHTML)

	// API routes (rate limited)
	r.Group(func(r chi.Router) {
		r.Use(rl.Handler)
		r.Post("/create", h.HandleCreate)
		r.Post("/read/{id:[0-9a-fA-F-]{36}}", h.HandleRead)
	})

	return r
}
