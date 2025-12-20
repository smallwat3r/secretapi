package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RedirectSlashes)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/robots.txt", h.HandleRobotsTXT)
	r.Get("/health", h.HandleHealth)

	fs := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	r.Group(func(r chi.Router) {
		r.Use(RateLimiter(DefaultRateLimitConfig()))
		r.Get("/", h.HandleIndexHTML)
		r.Route("/create", func(r chi.Router) {
			r.Post("/", h.HandleCreate)
		})
		r.Route("/read/{id:[0-9a-fA-F-]{36}}", func(r chi.Router) {
			r.Get("/", h.HandleIndexHTML)
			r.Post("/", h.HandleRead)
		})
	})

	return r
}
