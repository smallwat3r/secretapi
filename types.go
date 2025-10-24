package main

import "time"

type createReq struct {
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
	Expiry     string `json:"expiry"` // one of: 1h, 6h, 1day, 3days
}

type createRes struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type readReq struct {
	Passphrase string `json:"passphrase"`
}

type readRes struct {
	Secret string `json:"secret"`
}
