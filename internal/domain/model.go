package domain

import "time"

type CreateReq struct {
	Secret string `json:"secret"`
	Expiry string `json:"expiry"` // one of: 1h, 6h, 1d, 3d
}

type CreateRes struct {
	ID         string    `json:"id"`
	Passphrase string    `json:"passphrase"`
	ExpiresAt  time.Time `json:"expires_at"`
	URL        string    `json:"url"`
}

type ReadReq struct {
	Passphrase string `json:"passphrase"`
}

type ReadRes struct {
	Secret string `json:"secret"`
}
