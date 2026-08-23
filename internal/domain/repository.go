package domain

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const maxWatchRetries = 3

type SecretRepository interface {
	StoreSecret(ctx context.Context, id string, secret []byte, ttl time.Duration) error
	GetSecret(ctx context.Context, id string) ([]byte, error)
	DeleteSecret(ctx context.Context, id string) error
	IncrFailAndMaybeDelete(ctx context.Context, id string) (int64, error)
	Ping(ctx context.Context) error
}

type redisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository(rdb *redis.Client) SecretRepository {
	return &redisRepository{rdb: rdb}
}

func (r *redisRepository) Ping(ctx context.Context) error {
	return r.rdb.Ping(ctx).Err()
}

func (r *redisRepository) StoreSecret(
	ctx context.Context, id string, secret []byte, ttl time.Duration,
) error {
	key := redisKey(id)
	return r.rdb.Set(ctx, key, secret, ttl).Err()
}

func (r *redisRepository) GetSecret(ctx context.Context, id string) ([]byte, error) {
	key := redisKey(id)
	return r.rdb.Get(ctx, key).Bytes()
}

// DeleteSecret removes the secret and its attempts counter. The blob for an
// id is never overwritten, so a plain DEL is safe: the key either still holds
// the value we read or is already gone.
func (r *redisRepository) DeleteSecret(ctx context.Context, id string) error {
	return r.rdb.Del(ctx, redisKey(id), attemptsKey(id)).Err()
}

func (r *redisRepository) IncrFailAndMaybeDelete(ctx context.Context, id string) (int64, error) {
	key := redisKey(id)
	att := attemptsKey(id)
	var cnt *redis.IntCmd

	var err error
	for i := 0; i < maxWatchRetries; i++ {
		err = r.rdb.Watch(ctx, func(tx *redis.Tx) error {
			exists, err := tx.Exists(ctx, key).Result()
			if err != nil {
				return err
			}
			if exists == 0 {
				return nil // secret gone
			}

			ttl, err := tx.PTTL(ctx, key).Result()
			if err != nil {
				return err
			}

			// INCR attempts and align TTL
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				cnt = pipe.Incr(ctx, att)
				if ttl > 0 {
					pipe.PExpire(ctx, att, ttl)
				}
				return nil
			})
			if err != nil {
				return err
			}

			if cnt.Val() >= MaxReadAttempts {
				// delete secret and attempts counter
				_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
					pipe.Del(ctx, key, att)
					return nil
				})
				return err
			}
			return nil
		}, key, att)

		if !errors.Is(err, redis.TxFailedErr) {
			break
		}
		cnt = nil // reset for retry
	}

	if err != nil && !errors.Is(err, redis.TxFailedErr) {
		log.Printf("IncrFailAndMaybeDelete failed for id=%s: %v", id, err)
		return 0, err
	}

	if cnt == nil {
		return 0, nil
	}
	return cnt.Val(), nil
}

func redisKey(id string) string    { return "secret:" + id }
func attemptsKey(id string) string { return "secret:attempts:" + id }
