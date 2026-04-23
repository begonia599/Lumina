package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"mime"
	"os"
	"path"
	"strings"

	"lumina/internal/database"
)

var epubFontContentTypes = map[string]string{
	".eot":   "application/vnd.ms-fontobject",
	".otf":   "font/otf",
	".sfnt":  "font/sfnt",
	".ttc":   "font/collection",
	".ttf":   "font/ttf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
}

func openEPUBFromPath(sourcePath string) (*zip.Reader, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read epub: %w", err)
	}
	return zip.NewReader(bytes.NewReader(data), int64(len(data)))
}

func GetEPUBResource(ctx context.Context, userID, bookID int, resourcePath string) ([]byte, string, error) {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return nil, "", err
	}

	var format string
	var sourcePath *string
	err := database.Pool.QueryRow(ctx,
		`SELECT format, source_path FROM books WHERE id = $1 AND user_id = $2`,
		bookID, userID,
	).Scan(&format, &sourcePath)
	if err != nil || sourcePath == nil || *sourcePath == "" || format != "epub" {
		return nil, "", ErrNotFound
	}

	resourcePath = normalizeEPUBPath(strings.TrimPrefix(resourcePath, "/"))
	if resourcePath == "" || strings.Contains(resourcePath, "..") {
		return nil, "", fmt.Errorf("invalid resource path")
	}

	zr, err := openEPUBFromPath(*sourcePath)
	if err != nil {
		return nil, "", err
	}
	data, err := readZipEntry(zr, resourcePath)
	if err != nil {
		return nil, "", ErrNotFound
	}

	contentType := detectEPUBResourceContentType(resourcePath, data)
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return data, contentType, nil
	case strings.HasPrefix(contentType, "font/"),
		strings.HasPrefix(contentType, "application/font-"),
		strings.HasPrefix(contentType, "application/x-font-"),
		contentType == "application/vnd.ms-fontobject",
		contentType == "application/font-sfnt":
		return data, contentType, nil
	case contentType == "text/css":
		sanitized := sanitizeStylesheet(string(data), bookID, path.Dir(resourcePath), "")
		if sanitized == "" {
			return nil, "", fmt.Errorf("sanitized stylesheet is empty")
		}
		return []byte(sanitized), contentType, nil
	default:
		return nil, "", fmt.Errorf("unsupported resource type")
	}
}

func detectEPUBResourceContentType(name string, data []byte) string {
	ext := strings.ToLower(path.Ext(name))
	if contentType, ok := epubFontContentTypes[ext]; ok {
		return contentType
	}
	if ext == ".css" {
		return "text/css"
	}
	contentType := detectEPUBMime(name, data)
	if contentType != "application/octet-stream" {
		return contentType
	}
	if extType := mime.TypeByExtension(ext); extType != "" {
		return extType
	}
	return contentType
}
