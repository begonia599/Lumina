package handler

import (
	"errors"
	"net/http"
	"os"

	"lumina/internal/auth"
	"lumina/internal/httpx"
	"lumina/internal/middleware"

	"github.com/gin-gonic/gin"
)

// AuthHandler wires the Provider into HTTP endpoints.
type AuthHandler struct {
	provider        auth.Provider
	sessionLifetime int // seconds, for cookie Max-Age
	secureCookie    bool
}

// NewAuthHandler constructs an AuthHandler. sessionLifetimeSeconds must match
// the provider's session TTL so the cookie expires alongside the server record.
func NewAuthHandler(provider auth.Provider, sessionLifetimeSeconds int, secureCookie bool) *AuthHandler {
	return &AuthHandler{
		provider:        provider,
		sessionLifetime: sessionLifetimeSeconds,
		secureCookie:    secureCookie,
	}
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	User userDTO `json:"user"`
}

type userDTO struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// Register handles POST /api/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "请求格式错误")
		return
	}
	principal, token, err := h.provider.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.writeAuthError(c, err)
		return
	}
	h.setSessionCookie(c, string(token))
	c.JSON(http.StatusCreated, authResponse{User: userDTO{ID: principal.UserID, Username: principal.Username}})
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "请求格式错误")
		return
	}
	principal, token, err := h.provider.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.writeAuthError(c, err)
		return
	}
	h.setSessionCookie(c, string(token))
	c.JSON(http.StatusOK, authResponse{User: userDTO{ID: principal.UserID, Username: principal.Username}})
}

// Logout handles POST /api/auth/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	if cookie, err := c.Cookie(middleware.SessionCookieName); err == nil && cookie != "" {
		_ = h.provider.Logout(c.Request.Context(), auth.SessionToken(cookie))
	}
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

// Me handles GET /api/auth/me.
// Expects RequireAuth to have populated the Gin context.
func (h *AuthHandler) Me(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	c.JSON(http.StatusOK, authResponse{User: userDTO{ID: p.UserID, Username: p.Username}})
}

// setSessionCookie writes the HTTP-only session cookie (ADR-8 / ADR-9).
func (h *AuthHandler) setSessionCookie(c *gin.Context, token string) {
	// path="/" ensures the cookie is sent to all API and SPA routes.
	// SameSite=Lax is enforced below via c.SetSameSite to avoid depending on the net/http default.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.SessionCookieName, token, h.sessionLifetime, "/", "", h.secureCookie, true)
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", h.secureCookie, true)
}

// writeAuthError maps auth package errors to HTTP responses.
func (h *AuthHandler) writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "账号或密码错误")
	case errors.Is(err, auth.ErrRegistrationDisabled):
		httpx.Error(c, http.StatusForbidden, httpx.CodeForbidden, "注册已关闭")
	case errors.Is(err, auth.ErrUsernameTaken):
		httpx.Error(c, http.StatusConflict, httpx.CodeConflict, "用户名已被占用")
	case errors.Is(err, auth.ErrInvalidUsername):
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "用户名不合法（1-64 字符，不允许零宽/控制字符）")
	case errors.Is(err, auth.ErrInvalidPassword):
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "密码至少 8 位")
	default:
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
	}
}

// IsSecureCookieFromEnv reads the SESSION_COOKIE_SECURE env var (default false in dev).
func IsSecureCookieFromEnv() bool {
	v := os.Getenv("SESSION_COOKIE_SECURE")
	return v == "true" || v == "1"
}
