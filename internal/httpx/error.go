// Package httpx centralizes HTTP response conventions: unified error envelope,
// error code strings, and small helpers. Handlers MUST use httpx.Error instead
// of gin.H{"error": ...} to keep the API contract consistent (plan §2.5).
package httpx

import "github.com/gin-gonic/gin"

// Error code strings. The frontend can branch on `error.code`.
const (
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeValidation   = "VALIDATION"
	CodeConflict     = "CONFLICT"
	CodeRateLimited  = "RATE_LIMITED"
	CodeInternal     = "INTERNAL"
)

// ErrorBody is the JSON shape of every non-2xx response.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error writes the unified error envelope and aborts with the given status.
// Handlers should typically `return` immediately after calling this.
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}
