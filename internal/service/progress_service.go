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
		`SELECT id, user_id, book_id, chapter_idx, char_offset, percentage, updated_at
		 FROM reading_progress
		 WHERE book_id = $1 AND user_id = $2`,
		bookID, userID,
	).Scan(&p.ID, &p.UserID, &p.BookID, &p.ChapterIdx, &p.CharOffset, &p.Percentage, &p.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

// UpdateProgress upserts the reading progress for (userID, bookID).
// chapterIdx / charOffset authoritative; percentage is a derived UI hint.
func UpdateProgress(ctx context.Context, userID, bookID, chapterIdx, charOffset int, percentage float64) error {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return err
	}
	_, err := database.Pool.Exec(ctx,
		`INSERT INTO reading_progress (user_id, book_id, chapter_idx, char_offset, percentage, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (book_id, user_id) DO UPDATE SET
		   chapter_idx = EXCLUDED.chapter_idx,
		   char_offset = EXCLUDED.char_offset,
		   percentage  = EXCLUDED.percentage,
		   updated_at  = NOW()`,
		userID, bookID, chapterIdx, charOffset, percentage,
	)
	return err
}
