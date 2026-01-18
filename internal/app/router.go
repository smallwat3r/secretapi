package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router wraps chi router with cleanup capabilities.
type Router struct {
	handler     http.Handler
	rateLimiter *RateLimiterMiddleware
}

// ServeHTTP implements http.Handler.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.handler.ServeHTTP(w, r)
}

// Stop cleans up background goroutines.
func (rt *Router) Stop() {
	rt.rateLimiter.Stop()
}

// cacheControl wraps an http.Handler to add Cache-Control headers.
func cacheControl(h http.Handler, maxAge time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control",
			fmt.Sprintf("public, max-age=%d", int(maxAge.Seconds())))
		h.ServeHTTP(w, r)
	})
}

func NewRouter(h *Handler) *Router {
	r := chi.NewRouter()
	rl := NewRateLimiter(DefaultRateLimitConfig())

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RedirectSlashes)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(SecurityHeaders)

	r.Get("/robots.txt", h.HandleRobotsTXT)
	r.Get("/health", h.HandleHealth)

	fs := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*",
		http.StripPrefix("/static/", cacheControl(fs, 24*time.Hour)))

	r.Group(func(r chi.Router) {
		r.Use(rl.Handler)
		r.Get("/", h.HandleIndexHTML)
		r.Route("/create", func(r chi.Router) {
			r.Post("/", h.HandleCreate)
		})
		r.Route("/read/{id:[0-9a-fA-F-]{36}}", func(r chi.Router) {
			r.Get("/", h.HandleIndexHTML)
			r.Post("/", h.HandleRead)
		})
	})

	return &Router{
		handler:     r,
		rateLimiter: rl,
	}
}
