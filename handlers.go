package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Secret = strings.TrimSpace(req.Secret)
	req.Passphrase = strings.TrimSpace(req.Passphrase)
	if req.Secret == "" || req.Passphrase == "" {
		httpError(w, http.StatusBadRequest, "secret and passphrase are required")
		return
	}
	if !validatePassphrase(req.Passphrase) {
		httpError(w, http.StatusBadRequest, "passphrase must be at least 8 characters long and contain at least one letter and one digit")
		return
	}

	ttl, ok := parseExpiry(req.Expiry)
	if !ok {
		httpError(w, http.StatusBadRequest, "expiry must be one of: 1h, 6h, 1day, 3days")
		return
	}

	blob, err := encryptToBlob([]byte(req.Secret), req.Passphrase)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "encryption failed")
		return
	}

	id := uuid.NewString()
	key := redisKey(id)

	if err := rdb.Set(ctx, key, blob, ttl).Err(); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to store secret")
		return
	}

	expiresAt := time.Now().Add(ttl).UTC()
	writeJSON(w, http.StatusCreated, createRes{ID: id, ExpiresAt: expiresAt})
}

func handleRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httpError(w, http.StatusBadRequest, "missing id")
		return
	}
	var req readReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Passphrase = strings.TrimSpace(req.Passphrase)
	if req.Passphrase == "" {
		httpError(w, http.StatusBadRequest, "passphrase is required")
		return
	}

	key := redisKey(id)
	blob, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			httpError(w, http.StatusNotFound, "not found or expired")
			return
		}
		httpError(w, http.StatusInternalServerError, "failed to fetch secret")
		return
	}

	plaintext, err := decryptFromBlob(blob, req.Passphrase)
	if err != nil {
		// wrong passphrase
		incrFailAndMaybeDelete(id, key)
		httpError(w, http.StatusUnauthorized, "invalid passphrase or corrupted data")
		return
	}

	// successful decrypt
	delIfMatch(key, blob)
	// tidy up attempts counter
	_ = rdb.Del(ctx, attemptsKey(id)).Err()

	writeJSON(w, http.StatusOK, readRes{Secret: string(plaintext)})
}
