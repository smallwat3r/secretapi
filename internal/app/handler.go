package app

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"secretapi/internal/domain"
	"secretapi/pkg/utility"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
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
	if len(req.Secret) > 64*1024 { // 64 KB limit
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
		ttl = time.Hour * 24 // 1 day as a default
	} else {
		var ok bool
		ttl, ok = utility.ParseExpiry(req.Expiry)
		if !ok {
			utility.HttpError(w, http.StatusBadRequest, "expiry must be one of: 1h, 6h, 1day, 3days")
			return
		}
	}

	blob, err := utility.Encrypt([]byte(req.Secret), passcode)
	if err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "encryption failed")
		return
	}

	id := uuid.NewString()

	if err := h.repo.StoreSecret(id, blob, ttl); err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "failed to store secret")
		return
	}

	log.Printf("secret created: id=%s expiry=%s", id, ttl)

	expiresAt := time.Now().Add(ttl).UTC()

	readURL := &url.URL{
		Scheme: "http",
		Host:   r.Host,
		Path:   "/read/" + id,
	}
	if r.TLS != nil {
		readURL.Scheme = "https"
	}

	utility.WriteJSON(w, http.StatusCreated, domain.CreateRes{ID: id, Passcode: passcode, ExpiresAt: expiresAt, ReadURL: readURL.String()})
}

func (h *Handler) HandleRead(w http.ResponseWriter, r *http.Request) {
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

	blob, err := h.repo.GetSecret(id)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			utility.HttpError(w, http.StatusNotFound, "not found or expired")
			return
		}
		utility.HttpError(w, http.StatusInternalServerError, "failed to fetch secret")
		return
	}

	plaintext, err := utility.Decrypt(blob, passcode)
	if err != nil {
		// wrong passcode
		log.Printf("invalid passcode for secret: id=%s", id)
		attempts := h.repo.IncrFailAndMaybeDelete(id)
		utility.WriteJSON(w, http.StatusUnauthorized, domain.ReadRes{
			RemainingAttempts: utility.IntPtr(3 - int(attempts)),
		})
		return
	}

	// successful decrypt
	log.Printf("secret successfully read: id=%s", id)
	h.repo.DelIfMatch(id, blob)
	// tidy up attempts counter
	_ = h.repo.DeleteAttempts(id)

	format := r.URL.Query().Get("format")
	if format == "plain" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write(plaintext)
		return
	}

	utility.WriteJSON(w, http.StatusOK, domain.ReadRes{Secret: string(plaintext)})
}

func renderTemplate(w http.ResponseWriter, r *http.Request, name string, data any) {
	t, err := template.ParseFiles("web/templates/layout.html", "web/templates/"+name)
	if err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "failed to parse template")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = t.Execute(w, data)
	if err != nil {
		utility.HttpError(w, http.StatusInternalServerError, "failed to execute template")
	}
}

func (h *Handler) HandleReadHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	renderTemplate(w, r, "read.html", nil)
}

func (h *Handler) HandleCreateHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	renderTemplate(w, r, "create.html", nil)
}

func (h *Handler) HandleRobotsTXT(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/robots.txt")
}

func RateLimiter(next http.Handler) http.Handler {
	postLimiter := rate.NewLimiter(2, 5)  // 2 req a second with burst of 5
	getLimiter := rate.NewLimiter(10, 20) // 10 req a second with burst of 20
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var limiter *rate.Limiter
		switch r.Method {
		case http.MethodPost:
			limiter = postLimiter
		case http.MethodGet:
			limiter = getLimiter
		}

		if limiter != nil && !limiter.Allow() {
			utility.HttpError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}
