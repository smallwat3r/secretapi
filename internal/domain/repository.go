package domain

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type SecretRepository interface {
	StoreSecret(id string, secret []byte, ttl time.Duration) error
	GetSecret(id string) ([]byte, error)
	DelIfMatch(id string, old []byte)
	IncrFailAndMaybeDelete(id string)
	DeleteAttempts(id string) error
}

type redisRepository struct {
	rdb *redis.Client
	ctx context.Context
}

func NewRedisRepository(rdb *redis.Client) SecretRepository {
	return &redisRepository{
		rdb: rdb,
		ctx: context.Background(),
	}
}

func (r *redisRepository) StoreSecret(id string, secret []byte, ttl time.Duration) error {
	key := redisKey(id)
	return r.rdb.Set(r.ctx, key, secret, ttl).Err()
}

func (r *redisRepository) GetSecret(id string) ([]byte, error) {
	key := redisKey(id)
	return r.rdb.Get(r.ctx, key).Bytes()
}

func (r *redisRepository) DeleteAttempts(id string) error {
	key := attemptsKey(id)
	return r.rdb.Del(r.ctx, key).Err()
}

// deletes the key if its current value equals old.
func (r *redisRepository) DelIfMatch(id string, old []byte) {
	key := redisKey(id)
	_ = r.rdb.Watch(r.ctx, func(tx *redis.Tx) error {
		cur, err := tx.Get(r.ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return nil
		}
		if !bytes.Equal(cur, old) {
			return redis.TxFailedErr
		}
		_, err = tx.TxPipelined(r.ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(r.ctx, key)
			return nil
		})
		return err
	}, key)
}

// increments attempts (TTL-aligned) and deletes at >=3.
func (r *redisRepository) IncrFailAndMaybeDelete(id string) {
	key := redisKey(id)
	att := attemptsKey(id)
	_ = r.rdb.Watch(r.ctx, func(tx *redis.Tx) error {
		exists, err := tx.Exists(r.ctx, key).Result()
		if err != nil || exists == 0 {
			return nil // secret gone
		}

		ttl, _ := tx.PTTL(r.ctx, key).Result()
		var cnt *redis.IntCmd

		// INCR attempts and align TTL
		_, err = tx.TxPipelined(r.ctx, func(pipe redis.Pipeliner) error {
			cnt = pipe.Incr(r.ctx, att)
			if ttl > 0 {
				pipe.PExpire(r.ctx, att, ttl)
			}
			return nil
		})
		if err != nil {
			return nil
		}

		if cnt.Val() >= 3 {
			// delete secret and attempts counter
			_, _ = tx.TxPipelined(r.ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(r.ctx, key, att)
				return nil
			})
		}
		return nil
	}, key, att)
}

func redisKey(id string) string    { return "secret:" + id }
func attemptsKey(id string) string { return "secret:attempts:" + id }
