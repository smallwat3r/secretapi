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

	r.Group(func(r chi.Router) {
		r.Use(RateLimiter)
		r.Get("/", h.HandleCreateHTML)
		r.Route("/create", func(r chi.Router) {
			r.Post("/", h.HandleCreate)
		})
		r.Route("/read/{id:[0-9a-fA-F-]{36}}", func(r chi.Router) {
			r.Get("/", h.HandleReadHTML)
			r.Post("/", h.HandleRead)
		})
	})

	return r
}
