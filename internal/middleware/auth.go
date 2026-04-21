// Package middleware holds HTTP middleware shared by all routes.
package middleware

import (
	"errors"
	"net/http"

	"lumina/internal/auth"
	"lumina/internal/httpx"

	"github.com/gin-gonic/gin"
)

// SessionCookieName is the HTTP-only cookie carrying the session token.
const SessionCookieName = "lumina_session"

// ContextKeyPrincipal is the Gin context key for the authenticated Principal.
const ContextKeyPrincipal = "principal"

// ContextKeyUserID is a convenience key for handlers that only need the ID.
const ContextKeyUserID = "userID"

// RequireAuth returns middleware that rejects requests without a valid session.
func RequireAuth(provider auth.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(SessionCookieName)
		if err != nil || cookie == "" {
			httpx.Error(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "请先登录")
			c.Abort()
			return
		}
		principal, err := provider.Authenticate(c.Request.Context(), auth.SessionToken(cookie))
		if err != nil {
			if errors.Is(err, auth.ErrUnauthenticated) {
				httpx.Error(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "会话已失效，请重新登录")
			} else {
				httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
			}
			c.Abort()
			return
		}
		c.Set(ContextKeyPrincipal, principal)
		c.Set(ContextKeyUserID, principal.UserID)
		c.Next()
	}
}

// MustUserID extracts the authenticated user's ID from the Gin context.
// Panics if called outside a RequireAuth-protected route.
func MustUserID(c *gin.Context) int {
	return c.MustGet(ContextKeyUserID).(int)
}

// MustPrincipal extracts the Principal. See MustUserID.
func MustPrincipal(c *gin.Context) *auth.Principal {
	return c.MustGet(ContextKeyPrincipal).(*auth.Principal)
}
