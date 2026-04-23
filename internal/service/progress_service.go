package service

import (
	"context"

	"lumina/internal/database"
	"lumina/internal/model"
)

// GetProgress returns the reading progress for (userID, bookID).
func GetProgress(ctx context.Context, userID, bookID int) (*model.ReadingProgress, error) {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return nil, err
	}
	var p model.ReadingProgress
	err := database.Pool.QueryRow(ctx,
		`SELECT id, user_id, book_id, chapter_idx, char_offset, anchor, scroll_pct, percentage, updated_at
		 FROM reading_progress
		 WHERE book_id = $1 AND user_id = $2`,
		bookID, userID,
	).Scan(&p.ID, &p.UserID, &p.BookID, &p.ChapterIdx, &p.CharOffset, &p.Anchor, &p.ScrollPct, &p.Percentage, &p.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

// UpdateProgress upserts the reading progress for (userID, bookID).
// chapterIdx / charOffset authoritative for TXT; EPUB may also persist anchor/scrollPct.
func UpdateProgress(ctx context.Context, userID, bookID, chapterIdx, charOffset int, anchor *string, scrollPct *float64, percentage float64) error {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return err
	}
	_, err := database.Pool.Exec(ctx,
		`INSERT INTO reading_progress (user_id, book_id, chapter_idx, char_offset, anchor, scroll_pct, percentage, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 ON CONFLICT (book_id, user_id) DO UPDATE SET
		   chapter_idx = EXCLUDED.chapter_idx,
		   char_offset = EXCLUDED.char_offset,
		   anchor      = EXCLUDED.anchor,
		   scroll_pct  = EXCLUDED.scroll_pct,
		   percentage  = EXCLUDED.percentage,
		   updated_at  = NOW()`,
		userID, bookID, chapterIdx, charOffset, anchor, scrollPct, percentage,
	)
	return err
}
