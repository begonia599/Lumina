package model

import "time"

// Session is an authenticated session record backing the HTTP-only cookie.
// The session ID is 32 bytes of crypto/rand encoded as base64url (~43 chars).
type Session struct {
	ID        string    `json:"-"`
	UserID    int       `json:"-"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}
