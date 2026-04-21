package handler

import (
	"net/http"

	"lumina/internal/httpx"
	"lumina/internal/middleware"
	"lumina/internal/service"

	"github.com/gin-gonic/gin"
)

// GetSettings handles GET /api/settings.
func GetSettings(c *gin.Context) {
	userID := middleware.MustUserID(c)

	settings, err := service.GetAllSettings(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings handles PUT /api/settings.
func UpdateSettings(c *gin.Context) {
	userID := middleware.MustUserID(c)

	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "请求格式错误")
		return
	}

	if err := service.UpdateSettings(c.Request.Context(), userID, req); err != nil {
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
