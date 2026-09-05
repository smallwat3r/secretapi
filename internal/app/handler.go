package app

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/smallwat3r/secretapi/internal/domain"
	"github.com/smallwat3r/secretapi/internal/utility"

	"github.com/smallwat3r/secretapi/web"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo         domain.SecretRepository
	defaultTheme string
}

func NewHandler(repo domain.SecretRepository, defaultTheme string) *Handler {
	return &Handler{repo: repo, defaultTheme: defaultTheme}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	// Check Redis if ?redis=true is passed
	if r.URL.Query().Get("redis") == "true" {
		if err := h.repo.Ping(r.Context()); err != nil {
			log.Printf("health check failed: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("redis unavailable"))
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	utility.WriteJSON(w, http.StatusOK, domain.ConfigRes{
		MaxSecretSize:   domain.MaxSecretSize,
		MaxReadAttempts: domain.MaxReadAttempts,
		ExpiryOptions:   domain.ExpiryOptions,
		DefaultTheme:    h.defaultTheme,
	})
}

func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxRequestBodySize)

	var req domain.CreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			utility.HttpError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		utility.HttpError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Secret = strings.TrimSpace(req.Secret)
	if req.Secret == "" {
		utility.HttpError(w, http.StatusBadRequest, "secret is required")
		return
	}
	if len(req.Secret) > domain.MaxSecretSize {
		utility.HttpError(w, http.StatusRequestEntityTooLarge, "secret exceeds 64KB limit")
		return
	}

	passcode, err := utility.GeneratePasscode()
	if err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "passcode generation failed")
		return
	}

	var ttl time.Duration
	if req.Expiry == "" {
		ttl = domain.DefaultExpiry
	} else {
		var ok bool
		ttl, ok = utility.ParseExpiry(req.Expiry)
		if !ok {
			utility.HttpError(w, http.StatusBadRequest, "expiry must be one of: 1h, 6h, 1d, 3d")
			return
		}
	}

	blob, err := utility.Encrypt([]byte(req.Secret), passcode)
	if err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "encryption failed")
		return
	}

	id := uuid.NewString()

	if err := h.repo.StoreSecret(r.Context(), id, blob, ttl); err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "failed to store secret")
		return
	}

	log.Printf("secret created: id=%s expiry=%s", id, ttl)

	expiresAt := time.Now().Add(ttl).UTC()

	scheme := "http"
	if isHTTPS(r) {
		scheme = "https"
	}
	readURL := &url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   "/read/" + id,
	}

	utility.WriteJSON(w, http.StatusCreated, domain.CreateRes{
		ID:        id,
		Passcode:  passcode,
		ExpiresAt: expiresAt,
		ReadURL:   readURL.String(),
	})
}

func (h *Handler) HandleRead(w http.ResponseWriter, r *http.Request) {
	// Reject any request body - passcode is sent via header
	if r.ContentLength > 0 {
		utility.HttpError(w, http.StatusBadRequest, "request body not allowed")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		utility.HttpError(w, http.StatusBadRequest, "missing id")
		return
	}

	passcode := r.Header.Get("X-Passcode")
	if passcode == "" {
		utility.HttpError(w, http.StatusBadRequest, "passcode is required")
		return
	}

	blob, err := h.repo.GetSecret(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utility.HttpError(w, http.StatusNotFound, "not found or expired")
			return
		}
		utility.HttpError(w, http.StatusInternalServerError, "failed to fetch secret")
		return
	}

	plaintext, err := utility.Decrypt(blob, passcode)
	if err != nil {
		log.Printf("invalid passcode for secret: id=%s", id)
		attempts, err := h.repo.IncrFailAndMaybeDelete(r.Context(), id)
		if err != nil {
			log.Printf("failed to record attempt: id=%s err=%v", id, err)
		}
		utility.WriteJSON(w, http.StatusUnauthorized, domain.ReadRes{
			RemainingAttempts: utility.IntPtr(domain.MaxReadAttempts - int(attempts)),
		})
		return
	}

	// Delete before responding: if the key was already gone, a concurrent
	// request won the race and this one must not return the secret too.
	deleted, err := h.repo.DeleteSecret(r.Context(), id)
	if err != nil {
		log.Printf("failed to delete secret after read: id=%s err=%v", id, err)
		utility.HttpError(w, http.StatusInternalServerError, "failed to delete secret")
		return
	}
	if !deleted {
		utility.HttpError(w, http.StatusNotFound, "not found or expired")
		return
	}
	log.Printf("secret successfully read: id=%s", id)

	format := r.URL.Query().Get("format")
	if format == "plain" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(plaintext)
		return
	}

	utility.WriteJSON(w, http.StatusOK, domain.ReadRes{Secret: string(plaintext)})
}

func (h *Handler) HandleIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFileFS(w, r, web.FS, "static/dist/index.html")
}

func (h *Handler) HandleRobotsTXT(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, web.FS, "robots.txt")
}
