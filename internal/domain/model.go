package domain

import "time"

type CreateReq struct {
	Secret string `json:"secret"`
	Expiry string `json:"expiry"` // one of: 1h, 6h, 1d, 3d
}

type CreateRes struct {
	ID         string    `json:"id"`
	Passcode   string    `json:"passcode"`
	ExpiresAt  time.Time `json:"expires_at"`
	URL        string    `json:"url"`
}

type ReadReq struct {
	Passcode string `json:"passcode"`
}

type ReadRes struct {
	Secret string `json:"secret"`
}
