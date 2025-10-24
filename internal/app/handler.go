package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"secretapi/internal/domain"
	"secretapi/pkg/utility"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	repo domain.SecretRepository
}

func NewHandler(repo domain.SecretRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utility.HttpError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Secret = strings.TrimSpace(req.Secret)
	if req.Secret == "" {
		utility.HttpError(w, http.StatusBadRequest, "secret is required")
		return
	}

	passphrase := uuid.NewString()

	var ttl time.Duration
	if req.Expiry == "" {
		ttl = time.Hour * 24 // 1 day as a default
	} else {
		var ok bool
		ttl, ok = utility.ParseExpiry(req.Expiry)
		if !ok {
			utility.HttpError(w, http.StatusBadRequest, "expiry must be one of: 1h, 6h, 1day, 3days")
			return
		}
	}

	blob, err := utility.Encrypt([]byte(req.Secret), passphrase)
	if err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "encryption failed")
		return
	}

	id := uuid.NewString()

	if err := h.repo.StoreSecret(id, blob, ttl); err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "failed to store secret")
		return
	}

	expiresAt := time.Now().Add(ttl).UTC()

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	url := scheme + "://" + host + "/read/" + id + "/" + passphrase + "/"

	utility.WriteJSON(w, http.StatusCreated, domain.CreateRes{ID: id, Passphrase: passphrase, ExpiresAt: expiresAt, URL: url})
}

func (h *Handler) HandleRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utility.HttpError(w, http.StatusBadRequest, "missing id")
		return
	}
	passphrase := chi.URLParam(r, "passphrase")
	if passphrase == "" {
		utility.HttpError(w, http.StatusBadRequest, "passphrase is required")
		return
	}

	blob, err := h.repo.GetSecret(id)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			utility.HttpError(w, http.StatusNotFound, "not found or expired")
			return
		}
		utility.HttpError(w, http.StatusInternalServerError, "failed to fetch secret")
		return
	}

	plaintext, err := utility.Decrypt(blob, passphrase)
	if err != nil {
		// wrong passphrase
		h.repo.IncrFailAndMaybeDelete(id)
		utility.HttpError(w, http.StatusUnauthorized, "invalid passphrase or corrupted data")
		return
	}

	// successful decrypt
	h.repo.DelIfMatch(id, blob)
	// tidy up attempts counter
	_ = h.repo.DeleteAttempts(id)

	utility.WriteJSON(w, http.StatusOK, domain.ReadRes{Secret: string(plaintext)})
}
