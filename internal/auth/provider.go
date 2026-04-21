// Package auth defines the pluggable authentication layer (ADR-13).
//
// Business code depends only on the Provider interface. The first version
// ships a LocalAuthProvider (username+password, bcrypt, DB-backed sessions).
// Swapping in an SDK-based provider later touches only main.go.
package auth

import (
	"context"
	"errors"

	"lumina/internal/model"
)

// Principal is the authenticated subject handed to business code.
// Handlers read it from the Gin context under the "principal" key.
type Principal struct {
	UserID   int
	Username string
}

// SessionToken is the opaque string carried in the HTTP-only cookie.
// Implementations decide its internal format.
type SessionToken string

// Provider abstracts authentication. Any concrete implementation
// (local, OIDC, in-house SDK) must satisfy this interface.
type Provider interface {
	// Register creates a new user and starts a session.
	// Returns ErrRegistrationDisabled if the provider is configured to reject
	// new sign-ups, ErrUsernameTaken on collision, or validation errors.
	Register(ctx context.Context, username, password string) (*Principal, SessionToken, error)

	// Login verifies credentials and starts a session.
	// On any failure returns ErrInvalidCredentials (never leaks which of
	// username/password was wrong).
	Login(ctx context.Context, username, password string) (*Principal, SessionToken, error)

	// Logout revokes a session.
	// A call with an unknown / already-expired token is a no-op.
	Logout(ctx context.Context, token SessionToken) error

	// Authenticate validates a token and returns the Principal.
	// Returns ErrUnauthenticated for unknown / expired sessions.
	Authenticate(ctx context.Context, token SessionToken) (*Principal, error)

	// GetUser returns the full User record by ID.
	// Used by /api/auth/me.
	GetUser(ctx context.Context, userID int) (*model.User, error)
}

// Error values. Implementations MUST use these (optionally wrapped) so the
// middleware and handler layers can map them to stable HTTP responses.
var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUnauthenticated      = errors.New("unauthenticated")
	ErrRegistrationDisabled = errors.New("registration disabled")
	ErrUsernameTaken        = errors.New("username taken")
	ErrInvalidUsername      = errors.New("invalid username")
	ErrInvalidPassword      = errors.New("invalid password")
)
