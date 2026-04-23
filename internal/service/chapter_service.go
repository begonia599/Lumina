package service

import (
	"context"
	"fmt"
	"html"
	"os"
	"strings"

	"lumina/internal/database"
	"lumina/internal/model"
)

// ChapterContent is the DTO for GET /api/books/:id/chapters/:idx.
// Paragraphs are pre-split on the server so the frontend does not re-implement
// paragraph boundary logic (ADR-4).
type ChapterContent struct {
	ChapterIdx int      `json:"chapterIdx"`
	Title      string   `json:"title"`
	Format     string   `json:"format"`
	CharCount  int      `json:"charCount"`
	Paragraphs []string `json:"paragraphs,omitempty"`
	HTML       string   `json:"html,omitempty"`
	CSS        string   `json:"css,omitempty"`
	PrevIdx    *int     `json:"prevIdx,omitempty"`
	NextIdx    *int     `json:"nextIdx,omitempty"`
}

// GetChapters returns the chapter list (TOC) for a book owned by userID.
func GetChapters(ctx context.Context, userID, bookID int) ([]model.Chapter, error) {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return nil, err
	}
	rows, err := database.Pool.Query(ctx,
		`SELECT id, book_id, chapter_idx, title, start_pos, end_pos, COALESCE(content_ref, '')
		 FROM chapters WHERE book_id = $1
		 ORDER BY chapter_idx ASC`, bookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []model.Chapter
	for rows.Next() {
		var ch model.Chapter
		if err := rows.Scan(&ch.ID, &ch.BookID, &ch.ChapterIdx, &ch.Title, &ch.StartPos, &ch.EndPos, &ch.ContentRef); err != nil {
			return nil, err
		}
		ch.Title = cleanDisplayText(ch.Title)
		chapters = append(chapters, ch)
	}

	return chapters, nil
}

// GetChapterContent reads one chapter and returns a server-segmented DTO.
func GetChapterContent(ctx context.Context, userID, bookID, chapterIdx int) (*ChapterContent, error) {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return nil, err
	}

	var filePath string
	var sourcePath *string
	var format string
	err := database.Pool.QueryRow(ctx,
		`SELECT file_path, source_path, format FROM books WHERE id = $1 AND user_id = $2`,
		bookID, userID,
	).Scan(&filePath, &sourcePath, &format)
	if err != nil {
		return nil, ErrNotFound
	}

	var (
		title            string
		contentRef       *string
		startPos, endPos int
	)
	err = database.Pool.QueryRow(ctx,
		`SELECT title, start_pos, end_pos, content_ref FROM chapters
		 WHERE book_id = $1 AND chapter_idx = $2`,
		bookID, chapterIdx,
	).Scan(&title, &startPos, &endPos, &contentRef)
	if err != nil {
		return nil, ErrNotFound
	}

	// Neighbor indexes for prev/next navigation.
	var prevIdx, nextIdx *int
	if chapterIdx > 0 {
		v := chapterIdx - 1
		var exists int
		if err := database.Pool.QueryRow(ctx,
			`SELECT 1 FROM chapters WHERE book_id = $1 AND chapter_idx = $2`,
			bookID, v,
		).Scan(&exists); err == nil {
			prevIdx = &v
		}
	}
	{
		v := chapterIdx + 1
		var exists int
		if err := database.Pool.QueryRow(ctx,
			`SELECT 1 FROM chapters WHERE book_id = $1 AND chapter_idx = $2`,
			bookID, v,
		).Scan(&exists); err == nil {
			nextIdx = &v
		}
	}

	if format == "epub" {
		if sourcePath == nil || *sourcePath == "" || contentRef == nil || *contentRef == "" {
			return nil, fmt.Errorf("epub chapter source missing")
		}
		zr, err := openEPUBFromPath(*sourcePath)
		if err != nil {
			return nil, err
		}
		html, css, err := SanitizeChapterHTML(ctx, zr, bookID, chapterIdx, *contentRef)
		if err != nil {
			return nil, err
		}
		return &ChapterContent{
			ChapterIdx: chapterIdx,
			Title:      cleanDisplayText(title),
			Format:     "epub",
			CharCount:  max(0, endPos-startPos),
			HTML:       html,
			CSS:        css,
			PrevIdx:    prevIdx,
			NextIdx:    nextIdx,
		}, nil
	}

	// Read file (already UTF-8).
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	content := string(data)

	if startPos < 0 {
		startPos = 0
	}
	if endPos > len(content) {
		endPos = len(content)
	}
	if startPos >= endPos {
		return nil, fmt.Errorf("invalid chapter boundaries: start=%d end=%d", startPos, endPos)
	}

	raw := content[startPos:endPos]
	// Decode HTML entities + strip invisibles before splitting / counting.
	decoded := cleanDisplayText(raw)
	paragraphs := SplitParagraphs(decoded)
	title = cleanDisplayText(title)

	return &ChapterContent{
		ChapterIdx: chapterIdx,
		Title:      title,
		Format:     "txt",
		Paragraphs: paragraphs,
		CharCount:  len([]rune(decoded)),
		PrevIdx:    prevIdx,
		NextIdx:    nextIdx,
	}, nil
}

// SplitParagraphs splits raw chapter text into a clean paragraph list.
//
// Strategy: split on any line break, trim each, drop empty lines.
// Works well for Chinese novels where paragraphs are separated by \n or \r\n.
// The first line is assumed to be the chapter heading already stored as
// chapter.title, so we drop it if present to avoid duplicating the heading.
func SplitParagraphs(raw string) []string {
	lines := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' })
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if trim == "" {
			continue
		}
		out = append(out, trim)
	}
	// Drop the first paragraph if it is the chapter heading.
	// Heuristic: chapter heading lines are short (< 40 runes) and commonly
	// start with 第 / Chapter / 序 / 楔 / 引 / 前 / 后 / 尾 / 番.
	if len(out) > 0 && isLikelyChapterHeading(out[0]) {
		out = out[1:]
	}
	return out
}

func isLikelyChapterHeading(s string) bool {
	if len([]rune(s)) > 40 {
		return false
	}
	prefixes := []string{"第", "Chapter", "CHAPTER", "序", "楔", "引", "前言", "后记", "尾声", "番外"}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// SearchHit is one structured match returned by SearchBook.
// Frontend uses (chapterIdx, charOffset) to navigate and
// (previewHighlightStart/Length) to render inline highlighting.
type SearchHit struct {
	ChapterIdx             int    `json:"chapterIdx"`
	ChapterTitle           string `json:"chapterTitle"`
	ParagraphIdx           int    `json:"paragraphIdx"`
	CharOffset             int    `json:"charOffset"` // offset in runes from chapter start
	HitSeq                 *int   `json:"hitSeq,omitempty"`
	Preview                string `json:"preview"`
	PreviewHighlightStart  int    `json:"previewHighlightStart"`  // rune offset in preview
	PreviewHighlightLength int    `json:"previewHighlightLength"` // rune length of match
}

// SearchResponse is the /search envelope.
type SearchResponse struct {
	Query string      `json:"query"`
	Hits  []SearchHit `json:"hits"`
	Total int         `json:"total"`
}

// SearchBook performs a case-insensitive substring search across the full
// text of a book and returns up to MaxSearchResults structured matches.
const MaxSearchResults = 100

func SearchBook(ctx context.Context, userID, bookID int, query string) (*SearchResponse, error) {
	if query == "" {
		return &SearchResponse{Query: query, Hits: []SearchHit{}, Total: 0}, nil
	}
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return nil, err
	}

	var filePath string
	var format string
	err := database.Pool.QueryRow(ctx,
		`SELECT file_path, format FROM books WHERE id = $1 AND user_id = $2`,
		bookID, userID,
	).Scan(&filePath, &format)
	if err != nil {
		return nil, ErrNotFound
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	content := string(data)

	// Chapter boundaries for position → (chapterIdx, paragraphIdx, charOffset) mapping.
	chapters, err := GetChapters(ctx, userID, bookID)
	if err != nil {
		return nil, err
	}

	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)
	queryByteLen := len(lowerQuery)
	chapterHitSeq := make(map[int]int)

	hits := make([]SearchHit, 0, 16)
	searchFrom := 0
	for len(hits) < MaxSearchResults {
		rel := strings.Index(lowerContent[searchFrom:], lowerQuery)
		if rel == -1 {
			break
		}
		absPos := searchFrom + rel

		// Locate chapter.
		chIdx, chTitle, chStart := 0, "", 0
		for _, ch := range chapters {
			if absPos >= ch.StartPos && absPos < ch.EndPos {
				chIdx = ch.ChapterIdx
				chTitle = ch.Title
				chStart = ch.StartPos
				break
			}
		}

		// Paragraph index within chapter: count \n in [chStart, absPos).
		paraIdx := strings.Count(content[chStart:absPos], "\n")

		// Preview window — 60 chars of context on each side (byte-based OK,
		// we convert to rune offsets for highlight coordinates).
		pvStart := absPos - 60
		if pvStart < 0 {
			pvStart = 0
		}
		pvEnd := absPos + queryByteLen + 60
		if pvEnd > len(content) {
			pvEnd = len(content)
		}
		// Decode entities first so highlight offsets align with what the
		// frontend will render.
		rawPreview := content[pvStart:pvEnd]
		decodedPreview := cleanDisplayText(rawPreview)
		// The "before match" slice decoded → rune count gives the
		// highlight start in the decoded preview.
		decodedBefore := cleanDisplayText(content[pvStart:absPos])
		highlightStart := len([]rune(decodedBefore))
		matchStr := content[absPos : absPos+queryByteLen]
		highlightLen := len([]rune(cleanDisplayText(matchStr)))

		// charOffset (runes) from chapter start. Use decoded text so that
		// offsets match what the frontend's paragraph rune counts report.
		decodedChapterPrefix := cleanDisplayText(content[chStart:absPos])
		charOffset := len([]rune(decodedChapterPrefix))

		hit := SearchHit{
			ChapterIdx:             chIdx,
			ChapterTitle:           cleanDisplayText(chTitle),
			ParagraphIdx:           paraIdx,
			CharOffset:             charOffset,
			Preview:                decodedPreview,
			PreviewHighlightStart:  highlightStart,
			PreviewHighlightLength: highlightLen,
		}
		if format == "epub" {
			seq := chapterHitSeq[chIdx]
			hit.HitSeq = &seq
			hit.ParagraphIdx = 0
			chapterHitSeq[chIdx] = seq + 1
		}

		hits = append(hits, hit)

		searchFrom = absPos + queryByteLen
	}

	return &SearchResponse{
		Query: query,
		Hits:  hits,
		Total: len(hits),
	}, nil
}

// utf8RuneIndex converts a byte offset in s to a rune offset.
// byteOff is assumed to be at a valid rune boundary.
func utf8RuneIndex(s string, byteOff int) int {
	if byteOff <= 0 {
		return 0
	}
	if byteOff >= len(s) {
		return len([]rune(s))
	}
	return len([]rune(s[:byteOff]))
}

// cleanDisplayText normalizes a chunk of stored text for display:
//   - HTML entities decoded (&#8226; → "•", &amp; → "&", etc.). Many TXT
//     files scraped from the web carry these raw; without this fix users
//     would see literal "&#8226;" strings mid-sentence.
//   - BOM / non-breaking space / soft hyphen scrubbed.
//
// Safe to call on already-clean text (idempotent for visible characters).
func cleanDisplayText(s string) string {
	if s == "" {
		return s
	}
	out := html.UnescapeString(s)
	// Strip a handful of invisible characters commonly found in scraped TXT.
	replacer := strings.NewReplacer(
		"\uFEFF", "", // BOM
		"\u00AD", "", // soft hyphen
		"\u200B", "", // zero-width space
		"\u200C", "", // zero-width non-joiner
		"\u200D", "", // zero-width joiner
		"\u00A0", " ", // nbsp → regular space
	)
	return replacer.Replace(out)
}
