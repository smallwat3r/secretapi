package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
	"unicode"
)

func parseExpiry(s string) (time.Duration, bool) {
	switch s {
	case "1h":
		return time.Hour, true
	case "6h":
		return 6 * time.Hour, true
	case "1day":
		return 24 * time.Hour, true
	case "3days":
		return 72 * time.Hour, true
	default:
		return 0, false
	}
}

func validatePassphrase(p string) bool {
	if len(p) < 8 {
		return false
	}
	hasLetter := false
	hasDigit := false

	for _, r := range p {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	return hasLetter && hasDigit
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
