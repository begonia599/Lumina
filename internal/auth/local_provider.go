package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"lumina/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// LocalAuthProvider authenticates against the local `users` + `sessions` tables.
// Passwords are bcrypt-hashed (cost=12). Sessions are 32 random bytes, base64url.
type LocalAuthProvider struct {
	pool                *pgxpool.Pool
	sessionLifetime     time.Duration
	registrationEnabled bool
	bcryptCost          int
}

// LocalProviderOptions configures the LocalAuthProvider.
type LocalProviderOptions struct {
	// SessionLifetime defaults to 30 days when zero.
	SessionLifetime time.Duration
	// RegistrationEnabled toggles the sign-up endpoint (ADR-14).
	RegistrationEnabled bool
	// BcryptCost defaults to 12 when zero.
	BcryptCost int
}

// NewLocalProvider constructs a LocalAuthProvider.
func NewLocalProvider(pool *pgxpool.Pool, opts LocalProviderOptions) *LocalAuthProvider {
	if opts.SessionLifetime == 0 {
		opts.SessionLifetime = 30 * 24 * time.Hour
	}
	if opts.BcryptCost == 0 {
		opts.BcryptCost = 12
	}
	return &LocalAuthProvider{
		pool:                pool,
		sessionLifetime:     opts.SessionLifetime,
		registrationEnabled: opts.RegistrationEnabled,
		bcryptCost:          opts.BcryptCost,
	}
}

// SessionLifetime exposes the configured session TTL (for cookie Max-Age).
func (p *LocalAuthProvider) SessionLifetime() time.Duration {
	return p.sessionLifetime
}

// Register implements Provider.
func (p *LocalAuthProvider) Register(ctx context.Context, username, password string) (*Principal, SessionToken, error) {
	if !p.registrationEnabled {
		return nil, "", ErrRegistrationDisabled
	}

	name := NormalizeUsername(username)
	if err := ValidateUsername(name); err != nil {
		return nil, "", err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), p.bcryptCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	var userID int
	// Parameterized — username carries arbitrary Unicode safely (ADR-12).
	err = p.pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, provider)
		 VALUES ($1, $2, 'local')
		 RETURNING id`,
		name, string(hash),
	).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, "", ErrUsernameTaken
		}
		return nil, "", fmt.Errorf("insert user: %w", err)
	}

	token, err := p.createSession(ctx, userID)
	if err != nil {
		return nil, "", err
	}

	return &Principal{UserID: userID, Username: name}, token, nil
}

// Login implements Provider.
func (p *LocalAuthProvider) Login(ctx context.Context, username, password string) (*Principal, SessionToken, error) {
	name := NormalizeUsername(username)
	// Do not short-circuit on empty username — run bcrypt anyway to even out timings.

	var (
		userID int
		hash   string
	)
	err := p.pool.QueryRow(ctx,
		`SELECT id, password_hash FROM users
		 WHERE username = $1 AND provider = 'local'`,
		name,
	).Scan(&userID, &hash)

	// Always run bcrypt once to keep timing roughly constant whether or not
	// the account exists. On user-miss we compare against a dummy hash.
	const dummyHash = "$2a$12$abcdefghijklmnopqrstuuKlQi5lSy3YoWcQvFLXE4lXP9OYkHPlqG"
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, "", fmt.Errorf("lookup user: %w", err)
		}
		_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := p.createSession(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	return &Principal{UserID: userID, Username: name}, token, nil
}

// Logout implements Provider.
func (p *LocalAuthProvider) Logout(ctx context.Context, token SessionToken) error {
	if token == "" {
		return nil
	}
	_, err := p.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, string(token))
	return err
}

// Authenticate implements Provider.
func (p *LocalAuthProvider) Authenticate(ctx context.Context, token SessionToken) (*Principal, error) {
	if token == "" {
		return nil, ErrUnauthenticated
	}
	var (
		userID    int
		username  string
		expiresAt time.Time
	)
	err := p.pool.QueryRow(ctx,
		`SELECT s.user_id, u.username, s.expires_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.id = $1`,
		string(token),
	).Scan(&userID, &username, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthenticated
		}
		return nil, fmt.Errorf("lookup session: %w", err)
	}
	if time.Now().After(expiresAt) {
		// Clean up expired session opportunistically.
		_, _ = p.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, string(token))
		return nil, ErrUnauthenticated
	}
	return &Principal{UserID: userID, Username: username}, nil
}

// GetUser implements Provider.
func (p *LocalAuthProvider) GetUser(ctx context.Context, userID int) (*model.User, error) {
	var u model.User
	err := p.pool.QueryRow(ctx,
		`SELECT id, username, provider, external_id, created_at, password_hash
		 FROM users WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Username, &u.Provider, &u.ExternalID, &u.CreatedAt, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthenticated
		}
		return nil, err
	}
	return &u, nil
}

// createSession inserts a new session row and returns the opaque token.
func (p *LocalAuthProvider) createSession(ctx context.Context, userID int) (SessionToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().Add(p.sessionLifetime)

	_, err := p.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, userID, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return SessionToken(token), nil
}

// isUniqueViolation returns true for Postgres unique_violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	type sqlStateErr interface{ SQLState() string }
	var s sqlStateErr
	if errors.As(err, &s) {
		return s.SQLState() == "23505"
	}
	return false
}
