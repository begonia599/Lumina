package handler

import (
	"errors"
	"net/http"
	"strconv"

	"lumina/internal/httpx"
	"lumina/internal/middleware"
	"lumina/internal/service"

	"github.com/gin-gonic/gin"
)

// GetProgress handles GET /api/books/:id/progress.
func GetProgress(c *gin.Context) {
	userID := middleware.MustUserID(c)
	bookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	progress, err := service.GetProgress(c.Request.Context(), userID, bookID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "进度记录不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusOK, progress)
}

// UpdateProgress handles PUT /api/books/:id/progress.
func UpdateProgress(c *gin.Context) {
	userID := middleware.MustUserID(c)
	bookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	var req struct {
		ChapterIdx int      `json:"chapterIdx"`
		CharOffset int      `json:"charOffset"`
		Anchor     *string  `json:"anchor"`
		ScrollPct  *float64 `json:"scrollPct"`
		Percentage float64  `json:"percentage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "请求格式错误")
		return
	}
	if req.ChapterIdx < 0 || req.CharOffset < 0 {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "进度值不合法")
		return
	}
	if req.Percentage < 0 {
		req.Percentage = 0
	} else if req.Percentage > 1 {
		req.Percentage = 1
	}
	if req.ScrollPct != nil {
		if *req.ScrollPct < 0 {
			v := 0.0
			req.ScrollPct = &v
		} else if *req.ScrollPct > 1 {
			v := 1.0
			req.ScrollPct = &v
		}
	}

	if err := service.UpdateProgress(c.Request.Context(), userID, bookID, req.ChapterIdx, req.CharOffset, req.Anchor, req.ScrollPct, req.Percentage); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
