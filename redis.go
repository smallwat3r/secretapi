package main

import (
	"bytes"
	"errors"

	"github.com/redis/go-redis/v9"
)

// deletes the key if its current value equals old.
func delIfMatch(key string, old []byte) {
	_ = rdb.Watch(ctx, func(tx *redis.Tx) error {
		cur, err := tx.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return nil
		}
		if !bytes.Equal(cur, old) {
			return redis.TxFailedErr
		}
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			return nil
		})
		return err
	}, key)
}

// increments attempts (TTL-aligned) and deletes at >=3.
func incrFailAndMaybeDelete(id, key string) {
	att := attemptsKey(id)
	_ = rdb.Watch(ctx, func(tx *redis.Tx) error {
		exists, err := tx.Exists(ctx, key).Result()
		if err != nil || exists == 0 {
			return nil // secret gone
		}

		ttl, _ := tx.PTTL(ctx, key).Result()
		var cnt *redis.IntCmd

		// INCR attempts and align TTL
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			cnt = pipe.Incr(ctx, att)
			if ttl > 0 {
				pipe.PExpire(ctx, att, ttl)
			}
			return nil
		})
		if err != nil {
			return nil
		}

		if cnt.Val() >= 3 {
			// delete secret and attempts counter
			_, _ = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, key, att)
				return nil
			})
		}
		return nil
	}, key, att)
}

func redisKey(id string) string    { return "secret:" + id }
func attemptsKey(id string) string { return "secret:attempts:" + id }
