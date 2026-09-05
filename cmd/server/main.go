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

	opt.PoolSize = cfg.RedisPoolSize
	opt.MinIdleConns = cfg.RedisMinIdle
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	opt.PoolTimeout = 4 * time.Second

	rdb := redis.NewClient(opt)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	repo := domain.NewRedisRepository(rdb)
	handler := app.NewHandler(repo, cfg.DefaultTheme)

	secCfg := app.SecurityHeadersConfig{
		RequireHTTPS:  cfg.RequireHTTPS,
		CanonicalHost: cfg.CanonicalHost,
	}

	rlCfg := app.DefaultRateLimitConfig()
	rlCfg.TrustedProxyCIDR = cfg.TrustedProxyCIDR

	router := app.NewRouter(handler, rdb, secCfg, rlCfg)

	srv := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
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

	if err := rdb.Close(); err != nil {
		log.Printf("failed to close redis connection: %v", err)
	}

	log.Println("server exiting")
}
