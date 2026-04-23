package handler

import (
	"errors"
	"net/http"
	"strconv"

	"lumina/internal/httpx"
	"lumina/internal/middleware"
	"lumina/internal/model"
	"lumina/internal/service"

	"github.com/gin-gonic/gin"
)

// GetBookmarks handles GET /api/books/:id/bookmarks.
func GetBookmarks(c *gin.Context) {
	userID := middleware.MustUserID(c)
	bookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	bookmarks, err := service.GetBookmarks(c.Request.Context(), userID, bookID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	if bookmarks == nil {
		bookmarks = []model.Bookmark{}
	}

	c.JSON(http.StatusOK, bookmarks)
}

// CreateBookmark handles POST /api/books/:id/bookmarks.
func CreateBookmark(c *gin.Context) {
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
		Note       string   `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "请求格式错误")
		return
	}
	if req.ChapterIdx < 0 || req.CharOffset < 0 {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "位置值不合法")
		return
	}

	bookmark, err := service.CreateBookmark(c.Request.Context(), userID, bookID, req.ChapterIdx, req.CharOffset, req.Anchor, req.ScrollPct, req.Note)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusCreated, bookmark)
}

// DeleteBookmark handles DELETE /api/bookmarks/:id.
func DeleteBookmark(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书签 id")
		return
	}

	if err := service.DeleteBookmark(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书签不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateBookmark handles PATCH /api/bookmarks/:id — note only.
func UpdateBookmark(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书签 id")
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "请求格式错误")
		return
	}
	// Cap note length so a malicious client can't balloon rows.
	if len(req.Note) > 1000 {
		req.Note = req.Note[:1000]
	}

	bm, err := service.UpdateBookmarkNote(c.Request.Context(), userID, id, req.Note)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书签不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusOK, bm)
}
