package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/smallwat3r/secretapi/internal/app"
	"github.com/smallwat3r/secretapi/internal/config"
	"github.com/smallwat3r/secretapi/internal/domain"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to parse redis url: %v", err)
	}

	// Configure connection pool
	opt.PoolSize = cfg.RedisPoolSize
	opt.MinIdleConns = cfg.RedisMinIdle
	opt.DialTimeout = cfg.RedisDialTimeout
	opt.ReadTimeout = cfg.RedisReadTimeout
	opt.WriteTimeout = cfg.RedisWriteTimeout
	opt.PoolTimeout = cfg.RedisPoolTimeout

	rdb := redis.NewClient(opt)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	repo := domain.NewRedisRepository(rdb)
	handler := app.NewHandler(repo)

	r := app.NewRouter(handler)

	srv := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           r,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	go func() {
		log.Printf("listening on %s", cfg.ListenAddr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	r.Stop()

	if err := rdb.Close(); err != nil {
		log.Printf("failed to close redis connection: %v", err)
	}

	log.Println("server exiting")
}
