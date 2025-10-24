package main

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

var (
	rdb *redis.Client
	ctx = context.Background()
)

func main() {
	// redis config from env
	addr := getenv("REDIS_ADDR", "localhost:6379")
	pw := getenv("REDIS_PASSWORD", "")
	dbStr := getenv("REDIS_DB", "0")
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		db = 0
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pw,
		DB:       db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/health", handleHealth)
	r.Post("/create", handleCreate)
	r.Post("/read/{id}", handleRead)

	port := getenv("PORT", "8080")
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
