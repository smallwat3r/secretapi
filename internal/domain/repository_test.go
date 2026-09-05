package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRepo(t *testing.T) (*redisRepository, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return &redisRepository{rdb: rdb}, mr
}

func TestRepository_StoreGetDelete(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	if _, err := repo.GetSecret(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := repo.StoreSecret(ctx, "id", []byte("blob"), time.Hour); err != nil {
		t.Fatalf("store: %v", err)
	}
	got, err := repo.GetSecret(ctx, "id")
	if err != nil || string(got) != "blob" {
		t.Fatalf("get: got %q, %v", got, err)
	}

	deleted, err := repo.DeleteSecret(ctx, "id")
	if err != nil || !deleted {
		t.Fatalf("first delete: deleted=%v err=%v", deleted, err)
	}
	// A second delete must report the key as already gone, this is what
	// stops two concurrent readers both receiving the secret.
	deleted, err = repo.DeleteSecret(ctx, "id")
	if err != nil || deleted {
		t.Fatalf("second delete: deleted=%v err=%v", deleted, err)
	}
}

func TestRepository_IncrFailAndMaybeDelete(t *testing.T) {
	repo, mr := newTestRepo(t)
	ctx := context.Background()

	if err := repo.StoreSecret(ctx, "id", []byte("blob"), time.Hour); err != nil {
		t.Fatalf("store: %v", err)
	}

	for want := int64(1); want < MaxReadAttempts; want++ {
		n, err := repo.IncrFailAndMaybeDelete(ctx, "id")
		if err != nil || n != want {
			t.Fatalf("attempt %d: got %d, %v", want, n, err)
		}
		if !mr.Exists(redisKey("id")) {
			t.Fatalf("secret deleted too early after %d attempts", want)
		}
	}
	if ttl := mr.TTL(attemptsKey("id")); ttl <= 0 || ttl > time.Hour {
		t.Errorf("attempts counter TTL not aligned with secret: %v", ttl)
	}

	n, err := repo.IncrFailAndMaybeDelete(ctx, "id")
	if err != nil || n != MaxReadAttempts {
		t.Fatalf("final attempt: got %d, %v", n, err)
	}
	if mr.Exists(redisKey("id")) || mr.Exists(attemptsKey("id")) {
		t.Fatal("secret and counter should be deleted after max attempts")
	}

	// Attempts against a missing secret are not counted.
	n, err = repo.IncrFailAndMaybeDelete(ctx, "id")
	if err != nil || n != 0 {
		t.Fatalf("missing secret: got %d, %v", n, err)
	}
	if mr.Exists(attemptsKey("id")) {
		t.Fatal("counter must not be created for a missing secret")
	}
}
