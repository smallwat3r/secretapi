package main

import (
	"context"
	"log"
	"net/http"

	"github.com/smallwat3r/secretapi/internal/app"
	"github.com/smallwat3r/secretapi/internal/domain"
	"github.com/smallwat3r/secretapi/pkg/utility"

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

	r := app.NewRouter(handler)

	port := utility.Getenv("PORT", "8080")
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
