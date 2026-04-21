package model

import "time"

// User represents a registered account.
// Provider/ExternalID fields are reserved for future SDK-based auth (see ADR-6, §2.6).
type User struct {
	ID         int       `json:"id"`
	Username   string    `json:"username"`
	Provider   string    `json:"provider"`
	ExternalID *string   `json:"externalId,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`

	// PasswordHash is never serialized to clients.
	PasswordHash string `json:"-"`
}
