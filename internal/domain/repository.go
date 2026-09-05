package domain

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a secret does not exist or has expired.
var ErrNotFound = errors.New("secret not found")

type SecretRepository interface {
	StoreSecret(ctx context.Context, id string, secret []byte, ttl time.Duration) error
	GetSecret(ctx context.Context, id string) ([]byte, error)
	// DeleteSecret removes the secret and reports whether it was still present,
	// so a caller can detect a concurrent read and keep the read-once guarantee.
	DeleteSecret(ctx context.Context, id string) (bool, error)
	// IncrFailAndMaybeDelete records a failed passcode attempt and deletes the
	// secret once MaxReadAttempts is reached. It returns the attempt count, or 0
	// if the secret was already gone.
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
	return r.rdb.Set(ctx, redisKey(id), secret, ttl).Err()
}

func (r *redisRepository) GetSecret(ctx context.Context, id string) ([]byte, error) {
	b, err := r.rdb.Get(ctx, redisKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	return b, err
}

// DeleteSecret removes the secret and its attempts counter. The blob for an
// id is never overwritten, so a plain DEL is safe: the key either still holds
// the value we read or is already gone.
func (r *redisRepository) DeleteSecret(ctx context.Context, id string) (bool, error) {
	pipe := r.rdb.Pipeline()
	del := pipe.Del(ctx, redisKey(id))
	pipe.Del(ctx, attemptsKey(id))
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	return del.Val() == 1, nil
}

// incrFailScript runs atomically on the server, so no WATCH/retry loop is
// needed: it counts an attempt only against a live secret, keeps the counter's
// TTL aligned with the secret, and deletes both once the limit is hit.
var incrFailScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return 0 end
local n = redis.call('INCR', KEYS[2])
local ttl = redis.call('PTTL', KEYS[1])
if ttl > 0 then redis.call('PEXPIRE', KEYS[2], ttl) end
if n >= tonumber(ARGV[1]) then redis.call('DEL', KEYS[1], KEYS[2]) end
return n
`)

func (r *redisRepository) IncrFailAndMaybeDelete(ctx context.Context, id string) (int64, error) {
	keys := []string{redisKey(id), attemptsKey(id)}
	return incrFailScript.Run(ctx, r.rdb, keys, MaxReadAttempts).Int64()
}

func redisKey(id string) string    { return "secret:" + id }
func attemptsKey(id string) string { return "secret:attempts:" + id }
