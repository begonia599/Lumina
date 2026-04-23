package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lumina/internal/httpx"
	"lumina/internal/middleware"
	"lumina/internal/model"
	"lumina/internal/service"

	"github.com/gin-gonic/gin"
)

// UploadBook handles POST /api/books/upload.
func UploadBook(c *gin.Context) {
	userID := middleware.MustUserID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "未收到上传文件")
		return
	}
	defer file.Close()

	rawData, err := io.ReadAll(file)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, "读取上传文件失败")
		return
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	book, err := service.CreateBook(c.Request.Context(), userID, header.Filename, rawData, uploadDir)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"book": book})
}

// ListBooks handles GET /api/books.
func ListBooks(c *gin.Context) {
	userID := middleware.MustUserID(c)

	books, err := service.ListBooks(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	if books == nil {
		books = []model.BookWithProgress{}
	}

	c.JSON(http.StatusOK, books)
}

// GetBook handles GET /api/books/:id.
func GetBook(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	book, err := service.GetBook(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusOK, book)
}

// UpdateBook handles PATCH /api/books/:id — editable metadata only.
func UpdateBook(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	// Use pointer fields so "absent" and "empty string" are distinguishable.
	var req struct {
		Title       *string   `json:"title"`
		Author      *string   `json:"author"`
		Description *string   `json:"description"`
		Tags        *[]string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "请求格式错误")
		return
	}

	book, err := service.UpdateBook(c.Request.Context(), userID, id, service.BookUpdate{
		Title:       req.Title,
		Author:      req.Author,
		Description: req.Description,
		Tags:        req.Tags,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"book": book})
}

// DeleteBook handles DELETE /api/books/:id.
func DeleteBook(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	if err := service.DeleteBook(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// UploadCover handles POST /api/books/:id/cover (multipart, field "file").
func UploadCover(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "未收到上传图片")
		return
	}
	defer file.Close()

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	if _, err := service.SetCover(c.Request.Context(), userID, id, file, uploadDir); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, err.Error())
		return
	}

	// Return the freshly-updated book so the frontend can pick up hasCover.
	book, err := service.GetBook(c.Request.Context(), userID, id)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"book": book})
}

// DeleteCover handles DELETE /api/books/:id/cover.
func DeleteCover(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	if err := service.DeleteCover(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "书籍不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// GetCover handles GET /api/books/:id/cover — streams the image bytes.
func GetCover(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "非法的书籍 id")
		return
	}

	path, err := service.GetCoverPath(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "封面不存在")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}

	// Serve with a short cache so edits propagate quickly to other tabs.
	c.Header("Cache-Control", "private, max-age=60")

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		c.Header("Content-Type", "image/jpeg")
	case ".png":
		c.Header("Content-Type", "image/png")
	case ".webp":
		c.Header("Content-Type", "image/webp")
	}
	c.File(path)
}

// GetEPUBResource handles GET /api/books/:id/resources/*path.
func GetEPUBResource(c *gin.Context) {
	userID := middleware.MustUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, "invalid book id")
		return
	}

	resourcePath := strings.TrimPrefix(c.Param("path"), "/")
	data, contentType, err := service.GetEPUBResource(c.Request.Context(), userID, id, resourcePath)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "resource not found")
			return
		}
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidation, err.Error())
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, data)
}
