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
	req.Passphrase = strings.TrimSpace(req.Passphrase)
	if req.Secret == "" || req.Passphrase == "" {
		utility.HttpError(w, http.StatusBadRequest, "secret and passphrase are required")
		return
	}
	if !utility.ValidatePassphrase(req.Passphrase) {
		utility.HttpError(w, http.StatusBadRequest, "passphrase must be at least 8 characters long and contain at least one letter and one digit")
		return
	}

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

	blob, err := utility.Encrypt([]byte(req.Secret), req.Passphrase)
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
	utility.WriteJSON(w, http.StatusCreated, domain.CreateRes{ID: id, ExpiresAt: expiresAt})
}

func (h *Handler) HandleRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utility.HttpError(w, http.StatusBadRequest, "missing id")
		return
	}
	var req domain.ReadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utility.HttpError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Passphrase = strings.TrimSpace(req.Passphrase)
	if req.Passphrase == "" {
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

	plaintext, err := utility.Decrypt(blob, req.Passphrase)
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
