package domain

import (
	"bytes"
	"context"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type SecretRepository interface {
	StoreSecret(id string, secret []byte, ttl time.Duration) error
	GetSecret(id string) ([]byte, error)
	DelIfMatch(id string, old []byte) error
	IncrFailAndMaybeDelete(id string) (int64, error)
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

// DelIfMatch deletes the key if its current value equals old.
func (r *redisRepository) DelIfMatch(id string, old []byte) error {
	key := redisKey(id)
	err := r.rdb.Watch(r.ctx, func(tx *redis.Tx) error {
		cur, err := tx.Get(r.ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			return nil // key already gone
		}
		if err != nil {
			return err
		}
		if !bytes.Equal(cur, old) {
			return redis.TxFailedErr // value changed, abort
		}
		_, err = tx.TxPipelined(r.ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(r.ctx, key)
			return nil
		})
		return err
	}, key)

	if err != nil && !errors.Is(err, redis.TxFailedErr) {
		log.Printf("DelIfMatch failed for id=%s: %v", id, err)
		return err
	}
	return nil
}

func (r *redisRepository) IncrFailAndMaybeDelete(id string) (int64, error) {
	key := redisKey(id)
	att := attemptsKey(id)
	var cnt *redis.IntCmd

	err := r.rdb.Watch(r.ctx, func(tx *redis.Tx) error {
		exists, err := tx.Exists(r.ctx, key).Result()
		if err != nil {
			return err
		}
		if exists == 0 {
			return nil // secret gone
		}

		ttl, err := tx.PTTL(r.ctx, key).Result()
		if err != nil {
			return err
		}

		// INCR attempts and align TTL
		_, err = tx.TxPipelined(r.ctx, func(pipe redis.Pipeliner) error {
			cnt = pipe.Incr(r.ctx, att)
			if ttl > 0 {
				pipe.PExpire(r.ctx, att, ttl)
			}
			return nil
		})
		if err != nil {
			return err
		}

		if cnt.Val() >= MaxReadAttempts {
			// delete secret and attempts counter
			_, err = tx.TxPipelined(r.ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(r.ctx, key, att)
				return nil
			})
			return err
		}
		return nil
	}, key, att)

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
