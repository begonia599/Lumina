package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lumina/internal/database"
	"lumina/internal/model"

	"github.com/gabriel-vasile/mimetype"
)

// CreateBook processes an uploaded book file and stores it for the user.
func CreateBook(ctx context.Context, userID int, filename string, rawData []byte, uploadBaseDir string) (*model.Book, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".epub":
		return createBookFromEPUB(ctx, userID, filename, rawData, uploadBaseDir)
	case ".txt", "":
		return createBookFromTXT(ctx, userID, filename, rawData, uploadBaseDir)
	default:
		return nil, fmt.Errorf("unsupported file format: %s (only .txt / .epub)", filepath.Ext(filename))
	}
}

// createBookFromTXT processes an uploaded TXT file: detects encoding,
// converts to UTF-8, parses chapters, and stores everything in the database.
func createBookFromTXT(ctx context.Context, userID int, filename string, rawData []byte, uploadBaseDir string) (*model.Book, error) {
	// 1. Detect encoding and convert to UTF-8
	utf8Data, detectedEncoding, err := DetectAndConvert(rawData)
	if err != nil {
		return nil, fmt.Errorf("encoding conversion failed: %w", err)
	}
	content := string(utf8Data)

	// 2. Save UTF-8 file to disk — per-user subdirectory
	userDir := filepath.Join(uploadBaseDir, strconv.Itoa(userID))
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	safeFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), sanitizeFilename(filename))
	filePath := filepath.Join(userDir, safeFilename)

	if err := os.WriteFile(filePath, utf8Data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 3. Extract title from filename (remove .txt extension)
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	// 4. Parse chapters
	chapterInfos := ParseChapters(content)

	// 5. Insert book record (user-scoped, parameterized)
	var bookID int
	err = database.Pool.QueryRow(ctx,
		`INSERT INTO books (user_id, title, filename, file_path, file_size, encoding, chapter_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		userID, title, filename, filePath, len(utf8Data), detectedEncoding, len(chapterInfos),
	).Scan(&bookID)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to insert book: %w", err)
	}

	// 6. Insert chapters in batch
	chapters := BuildChapterModels(bookID, chapterInfos)
	if err := insertChapters(ctx, chapters); err != nil {
		log.Printf("[WARN] Failed to insert chapters for book %d: %v", bookID, err)
	}

	// 7. Initialize reading progress (user-scoped)
	_, err = database.Pool.Exec(ctx,
		`INSERT INTO reading_progress (user_id, book_id, chapter_idx, char_offset, percentage)
		 VALUES ($1, $2, 0, 0, 0)`,
		userID, bookID,
	)
	if err != nil {
		log.Printf("[WARN] Failed to initialize progress for book %d: %v", bookID, err)
	}

	book := &model.Book{
		ID:           bookID,
		UserID:       userID,
		Title:        title,
		Tags:         []string{},
		Format:       "txt",
		Filename:     filename,
		FilePath:     filePath,
		FileSize:     int64(len(utf8Data)),
		Encoding:     detectedEncoding,
		ChapterCount: len(chapterInfos),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	log.Printf("[Book] Created (user %d): %q (encoding: %s, chapters: %d)",
		userID, title, detectedEncoding, len(chapterInfos))
	return book, nil
}

func createBookFromEPUB(ctx context.Context, userID int, filename string, rawData []byte, uploadBaseDir string) (*model.Book, error) {
	parsed, err := ParseEPUB(rawData)
	if err != nil {
		return nil, fmt.Errorf("parse epub: %w", err)
	}
	if parsed.IsFixedLayout {
		return nil, fmt.Errorf("fixed-layout EPUB is not supported yet")
	}

	userDir := filepath.Join(uploadBaseDir, strconv.Itoa(userID))
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	baseName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), sanitizeFilename(strings.TrimSuffix(filename, filepath.Ext(filename))))
	sourcePath := filepath.Join(userDir, baseName+".epub")
	filePath := filepath.Join(userDir, baseName+".txt")

	if err := os.WriteFile(sourcePath, rawData, 0644); err != nil {
		return nil, fmt.Errorf("failed to save epub: %w", err)
	}
	if err := os.WriteFile(filePath, []byte(parsed.PlainText), 0644); err != nil {
		_ = os.Remove(sourcePath)
		return nil, fmt.Errorf("failed to save epub index: %w", err)
	}

	title := parsed.Title
	if title == "" {
		title = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	var bookID int
	err = database.Pool.QueryRow(ctx,
		`INSERT INTO books (
			user_id, title, author, description, format, filename, file_path, source_path, file_size, encoding, chapter_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		userID, title, parsed.Author, parsed.Description, "epub", filename, filePath, sourcePath, len(rawData), "UTF-8", len(parsed.Chapters),
	).Scan(&bookID)
	if err != nil {
		_ = os.Remove(sourcePath)
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to insert book: %w", err)
	}

	chapters := make([]model.Chapter, 0, len(parsed.Chapters))
	for idx, ch := range parsed.Chapters {
		chapters = append(chapters, model.Chapter{
			BookID:     bookID,
			ChapterIdx: idx,
			Title:      ch.Title,
			StartPos:   ch.StartPos,
			EndPos:     ch.EndPos,
			ContentRef: ch.ContentRef,
		})
	}
	if err := insertChapters(ctx, chapters); err != nil {
		log.Printf("[WARN] Failed to insert EPUB chapters for book %d: %v", bookID, err)
	}

	_, err = database.Pool.Exec(ctx,
		`INSERT INTO reading_progress (user_id, book_id, chapter_idx, char_offset, scroll_pct, percentage)
		 VALUES ($1, $2, 0, 0, 0, 0)`,
		userID, bookID,
	)
	if err != nil {
		log.Printf("[WARN] Failed to initialize progress for EPUB book %d: %v", bookID, err)
	}

	var coverPath string
	if len(parsed.CoverData) > 0 {
		savedCoverPath, err := saveBookCoverBytes(uploadBaseDir, userID, bookID, parsed.CoverData, parsed.CoverMIME)
		if err != nil {
			log.Printf("[WARN] Failed to save EPUB cover for book %d: %v", bookID, err)
		} else {
			if _, err := database.Pool.Exec(ctx,
				`UPDATE books SET cover_path = $1, updated_at = NOW()
				 WHERE id = $2 AND user_id = $3`,
				savedCoverPath, bookID, userID,
			); err != nil {
				_ = os.Remove(savedCoverPath)
				log.Printf("[WARN] Failed to persist EPUB cover path for book %d: %v", bookID, err)
			} else {
				coverPath = savedCoverPath
			}
		}
	}

	return &model.Book{
		ID:           bookID,
		UserID:       userID,
		Title:        title,
		Author:       parsed.Author,
		Description:  parsed.Description,
		Tags:         []string{},
		Format:       "epub",
		Filename:     filename,
		FilePath:     filePath,
		SourcePath:   sourcePath,
		CoverPath:    coverPath,
		FileSize:     int64(len(rawData)),
		Encoding:     "UTF-8",
		ChapterCount: len(parsed.Chapters),
		HasCover:     coverPath != "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// bookColumns is the SELECT expression shared by ListBooks / GetBook so any
// column addition only changes one line.
const bookColumns = `b.id, b.user_id, b.title, b.author, b.description, b.tags,
b.format, b.filename, b.file_path, b.source_path, b.cover_path, b.file_size, b.encoding,
b.chapter_count, b.created_at, b.updated_at`

func scanBook(rows interface {
	Scan(dest ...any) error
}, b *model.Book) error {
	var coverPath *string
	var sourcePath *string
	err := rows.Scan(
		&b.ID, &b.UserID, &b.Title, &b.Author, &b.Description, &b.Tags,
		&b.Format, &b.Filename, &b.FilePath, &sourcePath, &coverPath, &b.FileSize, &b.Encoding,
		&b.ChapterCount, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if sourcePath != nil {
		b.SourcePath = *sourcePath
	}
	if coverPath != nil {
		b.CoverPath = *coverPath
		b.HasCover = true
	}
	if b.Tags == nil {
		b.Tags = []string{}
	}
	return nil
}

// ListBooks returns all books owned by userID with their reading progress.
func ListBooks(ctx context.Context, userID int) ([]model.BookWithProgress, error) {
	rows, err := database.Pool.Query(ctx,
		`SELECT `+bookColumns+`,
		        rp.id, rp.chapter_idx, rp.char_offset, rp.anchor, rp.scroll_pct, rp.percentage, rp.updated_at
		 FROM books b
		 LEFT JOIN reading_progress rp
		   ON rp.book_id = b.id AND rp.user_id = b.user_id
		 WHERE b.user_id = $1
		 ORDER BY COALESCE(rp.updated_at, b.updated_at) DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []model.BookWithProgress
	for rows.Next() {
		var b model.BookWithProgress
		var coverPath *string
		var sourcePath *string
		var rpID *int
		var rpChapter *int
		var rpOffset *int
		var rpAnchor *string
		var rpScrollPct *float64
		var rpPercent *float64
		var rpUpdated *time.Time

		err := rows.Scan(
			&b.ID, &b.UserID, &b.Title, &b.Author, &b.Description, &b.Tags,
			&b.Format, &b.Filename, &b.FilePath, &sourcePath, &coverPath, &b.FileSize, &b.Encoding,
			&b.ChapterCount, &b.CreatedAt, &b.UpdatedAt,
			&rpID, &rpChapter, &rpOffset, &rpAnchor, &rpScrollPct, &rpPercent, &rpUpdated,
		)
		if err != nil {
			return nil, err
		}
		if sourcePath != nil {
			b.SourcePath = *sourcePath
		}
		if coverPath != nil {
			b.CoverPath = *coverPath
			b.HasCover = true
		}
		if b.Tags == nil {
			b.Tags = []string{}
		}

		if rpID != nil {
			b.Progress = &model.ReadingProgress{
				ID:         *rpID,
				UserID:     userID,
				BookID:     b.ID,
				ChapterIdx: *rpChapter,
				CharOffset: *rpOffset,
				Anchor:     rpAnchor,
				ScrollPct:  rpScrollPct,
				Percentage: *rpPercent,
				UpdatedAt:  *rpUpdated,
			}
		}

		books = append(books, b)
	}

	return books, nil
}

// GetBook returns a single book owned by userID.
func GetBook(ctx context.Context, userID, bookID int) (*model.Book, error) {
	var b model.Book
	row := database.Pool.QueryRow(ctx,
		`SELECT `+bookColumns+` FROM books b WHERE b.id = $1 AND b.user_id = $2`,
		bookID, userID,
	)
	if err := scanBook(row, &b); err != nil {
		return nil, ErrNotFound
	}
	return &b, nil
}

// BookUpdate carries the editable subset of book metadata.
// Any field set to non-nil overwrites the stored value; nil fields are left alone.
type BookUpdate struct {
	Title       *string
	Author      *string
	Description *string
	Tags        *[]string
}

// UpdateBook applies the given partial update to (userID, bookID).
// Returns the full post-update Book.
func UpdateBook(ctx context.Context, userID, bookID int, upd BookUpdate) (*model.Book, error) {
	// Build dynamic SET clause using parameterized placeholders only.
	// (ADR-12: never interpolate user-supplied values into SQL strings.)
	sets := make([]string, 0, 4)
	args := make([]any, 0, 6)
	idx := 1

	if upd.Title != nil {
		t := strings.TrimSpace(*upd.Title)
		if t == "" {
			return nil, fmt.Errorf("title cannot be empty")
		}
		if len([]rune(t)) > 200 {
			return nil, fmt.Errorf("title too long")
		}
		sets = append(sets, fmt.Sprintf("title = $%d", idx))
		args = append(args, t)
		idx++
	}
	if upd.Author != nil {
		a := strings.TrimSpace(*upd.Author)
		if len([]rune(a)) > 80 {
			return nil, fmt.Errorf("author too long")
		}
		sets = append(sets, fmt.Sprintf("author = $%d", idx))
		args = append(args, a)
		idx++
	}
	if upd.Description != nil {
		d := strings.TrimSpace(*upd.Description)
		if len([]rune(d)) > 2000 {
			return nil, fmt.Errorf("description too long")
		}
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, d)
		idx++
	}
	if upd.Tags != nil {
		clean := cleanTags(*upd.Tags)
		sets = append(sets, fmt.Sprintf("tags = $%d", idx))
		args = append(args, clean)
		idx++
	}

	if len(sets) == 0 {
		return GetBook(ctx, userID, bookID)
	}

	sets = append(sets, "updated_at = NOW()")
	// NOTE: only `sets` (join of SET fragments built from literal SQL) is
	// interpolated; all values are passed as parameters. Safe.
	sql := "UPDATE books SET " + strings.Join(sets, ", ") +
		fmt.Sprintf(" WHERE id = $%d AND user_id = $%d", idx, idx+1)
	args = append(args, bookID, userID)

	tag, err := database.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("update book: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return GetBook(ctx, userID, bookID)
}

// cleanTags dedupes, trims, and size-limits a tag list.
func cleanTags(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len([]rune(t)) > 32 {
			t = string([]rune(t)[:32])
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= 20 {
			break
		}
	}
	return out
}

// ---------- Cover image ----------

// allowedCoverTypes maps detected MIME → file extension.
var allowedCoverTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// maxCoverBytes caps cover upload size (2 MB).
const maxCoverBytes = 2 * 1024 * 1024

// SetCover saves an uploaded image as the cover for (userID, bookID).
// Returns the new relative cover path.
func SetCover(ctx context.Context, userID, bookID int, file multipart.File, uploadBaseDir string) (string, error) {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return "", err
	}

	// Read with a hard size cap.
	limited := &io.LimitedReader{R: file, N: maxCoverBytes + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read cover: %w", err)
	}
	if int64(len(data)) > maxCoverBytes {
		return "", fmt.Errorf("cover exceeds 2MB")
	}

	// Detect MIME from bytes (not from filename — untrusted).
	mime := mimetype.Detect(data)
	coverPath, err := saveBookCoverBytes(uploadBaseDir, userID, bookID, data, mime.String())
	if err != nil {
		return "", err
	}

	_, err = database.Pool.Exec(ctx,
		`UPDATE books SET cover_path = $1, updated_at = NOW()
		 WHERE id = $2 AND user_id = $3`,
		coverPath, bookID, userID)
	if err != nil {
		os.Remove(coverPath)
		return "", fmt.Errorf("update cover path: %w", err)
	}

	return coverPath, nil
}

func saveBookCoverBytes(uploadBaseDir string, userID, bookID int, data []byte, mimeType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("cover data is empty")
	}

	if mimeType == "" {
		mimeType = mimetype.Detect(data).String()
	}
	ext, ok := allowedCoverTypes[mimeType]
	if !ok {
		return "", fmt.Errorf("unsupported image type: %s (only JPEG/PNG/WebP allowed)", mimeType)
	}

	coverDir := filepath.Join(uploadBaseDir, strconv.Itoa(userID), "covers")
	if err := os.MkdirAll(coverDir, 0755); err != nil {
		return "", fmt.Errorf("create cover dir: %w", err)
	}

	for _, oldExt := range []string{".jpg", ".png", ".webp"} {
		oldPath := filepath.Join(coverDir, fmt.Sprintf("book_%d%s", bookID, oldExt))
		_ = os.Remove(oldPath)
	}

	coverPath := filepath.Join(coverDir, fmt.Sprintf("book_%d%s", bookID, ext))
	if err := os.WriteFile(coverPath, data, 0644); err != nil {
		return "", fmt.Errorf("save cover: %w", err)
	}
	return coverPath, nil
}

// DeleteCover clears the cover for (userID, bookID), reverting to the
// procedural cover on the frontend. The image file on disk is removed too.
func DeleteCover(ctx context.Context, userID, bookID int) error {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return err
	}
	var oldPath *string
	err := database.Pool.QueryRow(ctx,
		`UPDATE books SET cover_path = NULL, updated_at = NOW()
		 WHERE id = $1 AND user_id = $2 RETURNING cover_path`,
		bookID, userID,
	).Scan(&oldPath)
	if err != nil {
		// If RETURNING returned nothing the book is gone; treat as not-found.
		return ErrNotFound
	}
	// Note: the returned cover_path is AFTER the UPDATE so it'll always be NULL.
	// Instead we scan for old covers by name and drop them.
	coverDir := filepath.Join(uploadBaseDirFromEnv(), strconv.Itoa(userID), "covers")
	for _, oldExt := range []string{".jpg", ".png", ".webp"} {
		p := filepath.Join(coverDir, fmt.Sprintf("book_%d%s", bookID, oldExt))
		_ = os.Remove(p)
	}
	return nil
}

// GetCoverPath returns the filesystem path of the stored cover for (userID, bookID).
// Returns ErrNotFound if the book has no uploaded cover.
func GetCoverPath(ctx context.Context, userID, bookID int) (string, error) {
	var path *string
	err := database.Pool.QueryRow(ctx,
		`SELECT cover_path FROM books WHERE id = $1 AND user_id = $2`,
		bookID, userID,
	).Scan(&path)
	if err != nil || path == nil || *path == "" {
		return "", ErrNotFound
	}
	return *path, nil
}

// uploadBaseDirFromEnv reads the upload base directory from env.
// Kept here to avoid circular imports with handler.
func uploadBaseDirFromEnv() string {
	v := os.Getenv("UPLOAD_DIR")
	if v == "" {
		v = "./uploads"
	}
	return v
}

// ---------- Delete book (cover cleanup added) ----------

// DeleteBook removes a book, its text file, its cover file, and all associated
// data (cascaded by FK). Authorization: book must belong to userID.
func DeleteBook(ctx context.Context, userID, bookID int) error {
	var filePath string
	var sourcePath *string
	var coverPath *string
	err := database.Pool.QueryRow(ctx,
		`DELETE FROM books WHERE id = $1 AND user_id = $2
		 RETURNING file_path, source_path, cover_path`,
		bookID, userID,
	).Scan(&filePath, &sourcePath, &coverPath)
	if err != nil {
		return ErrNotFound
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("[WARN] Failed to delete text file %s: %v", filePath, err)
	}
	if coverPath != nil && *coverPath != "" {
		if err := os.Remove(*coverPath); err != nil && !os.IsNotExist(err) {
			log.Printf("[WARN] Failed to delete cover file %s: %v", *coverPath, err)
		}
	}
	if sourcePath != nil && *sourcePath != "" {
		if err := os.Remove(*sourcePath); err != nil && !os.IsNotExist(err) {
			log.Printf("[WARN] Failed to delete EPUB file %s: %v", *sourcePath, err)
		}
	}

	log.Printf("[Book] Deleted book ID %d (user %d)", bookID, userID)
	return nil
}

// assertBookOwned returns ErrNotFound if bookID is not owned by userID.
func assertBookOwned(ctx context.Context, userID, bookID int) error {
	var owned int
	err := database.Pool.QueryRow(ctx,
		`SELECT 1 FROM books WHERE id = $1 AND user_id = $2`,
		bookID, userID,
	).Scan(&owned)
	if err != nil {
		return ErrNotFound
	}
	return nil
}

// insertChapters batch inserts chapters for a book.
func insertChapters(ctx context.Context, chapters []model.Chapter) error {
	if len(chapters) == 0 {
		return nil
	}

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, ch := range chapters {
		_, err := tx.Exec(ctx,
			`INSERT INTO chapters (book_id, chapter_idx, title, start_pos, end_pos, content_ref)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			ch.BookID, ch.ChapterIdx, ch.Title, ch.StartPos, ch.EndPos, ch.ContentRef,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// sanitizeFilename removes filesystem-dangerous characters from user filenames.
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"..", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
