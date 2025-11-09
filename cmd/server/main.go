package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smallwat3r/secretapi/internal/app"
	"github.com/smallwat3r/secretapi/internal/domain"
	"github.com/smallwat3r/secretapi/internal/utility"

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
	listenAddr := ":" + port

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}

	go func() {
		log.Printf("listening on %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown: ", err)
	}

	log.Println("server exiting")
}
