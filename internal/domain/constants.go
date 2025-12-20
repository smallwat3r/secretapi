package domain

import "time"

const (
	// MaxSecretSize is the maximum allowed size for a secret (64 KB).
	MaxSecretSize = 64 * 1024

	// MaxReadAttempts is the maximum number of incorrect passcode attempts
	// before a secret is automatically deleted.
	MaxReadAttempts = 3

	// DefaultExpiry is the default TTL for secrets when no expiry is specified.
	DefaultExpiry = 24 * time.Hour
)
