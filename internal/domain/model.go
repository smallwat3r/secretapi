package domain

import "time"

type CreateReq struct {
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
	Expiry     string `json:"expiry"` // one of: 1h, 6h, 1day, 3days
}

type CreateRes struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ReadReq struct {
	Passphrase string `json:"passphrase"`
}

type ReadRes struct {
	Secret string `json:"secret"`
}
