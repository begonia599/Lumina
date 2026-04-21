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

// GetChapters handles GET /api/books/:id/chapters.
func GetChapters(c *gin.Context) {
	userID := middleware.MustUserID(c)
	bookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	chapters, err := service.GetChapters(c.Request.Context(), userID, bookID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusOK, chapters)
}

// GetChapterContent handles GET /api/books/:id/chapters/:idx.
func GetChapterContent(c *gin.Context) {
	userID := middleware.MustUserID(c)
	bookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}
	chapterIdx, err := strconv.Atoi(c.Param("idx"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的章节序号")
		return
	}

	content, err := service.GetChapterContent(c.Request.Context(), userID, bookID, chapterIdx)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "章节不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusOK, content)
}

// SearchBook handles GET /api/books/:id/search?q=.
func SearchBook(c *gin.Context) {
	userID := middleware.MustUserID(c)
	bookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	query := c.Query("q")
	if query == "" {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "搜索关键词必填")
		return
	}

	resp, err := service.SearchBook(c.Request.Context(), userID, bookID, query)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusOK, resp)
}
