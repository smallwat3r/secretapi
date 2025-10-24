package main

import (
	"context"
	"log"
	"net/http"

	"secretapi/internal/app"
	"secretapi/internal/domain"
	"secretapi/pkg/utility"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func main() {
	redisURL := utility.Getenv("REDIS_URL", "redis://localhost:6379/0")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("failed to parse redis url: %v", err)
	}

	rdb := redis.NewClient(opt)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	repo := domain.NewRedisRepository(rdb)
	handler := app.NewHandler(repo)

	r := newRouter(handler)

	port := utility.Getenv("PORT", "8080")
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func newRouter(h *app.Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/health/", h.HandleHealth)
	r.Post("/create/", h.HandleCreate)
	r.Post("/read/{id}/", h.HandleRead)
	return r
}
