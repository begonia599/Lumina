package service

import (
	"context"

	"lumina/internal/database"
	"lumina/internal/model"
)

// GetBookmarks returns bookmarks for (userID, bookID).
func GetBookmarks(ctx context.Context, userID, bookID int) ([]model.Bookmark, error) {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return nil, err
	}
	rows, err := database.Pool.Query(ctx,
		`SELECT id, user_id, book_id, chapter_idx, char_offset, anchor, scroll_pct, note, created_at
		 FROM bookmarks
		 WHERE book_id = $1 AND user_id = $2
		 ORDER BY created_at DESC`,
		bookID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []model.Bookmark
	for rows.Next() {
		var bm model.Bookmark
		if err := rows.Scan(&bm.ID, &bm.UserID, &bm.BookID, &bm.ChapterIdx, &bm.CharOffset, &bm.Anchor, &bm.ScrollPct, &bm.Note, &bm.CreatedAt); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, bm)
	}

	return bookmarks, nil
}

// CreateBookmark adds a new bookmark for (userID, bookID).
func CreateBookmark(ctx context.Context, userID, bookID, chapterIdx, charOffset int, anchor *string, scrollPct *float64, note string) (*model.Bookmark, error) {
	if err := assertBookOwned(ctx, userID, bookID); err != nil {
		return nil, err
	}
	var bm model.Bookmark
	err := database.Pool.QueryRow(ctx,
		`INSERT INTO bookmarks (user_id, book_id, chapter_idx, char_offset, anchor, scroll_pct, note)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, user_id, book_id, chapter_idx, char_offset, anchor, scroll_pct, note, created_at`,
		userID, bookID, chapterIdx, charOffset, anchor, scrollPct, note,
	).Scan(&bm.ID, &bm.UserID, &bm.BookID, &bm.ChapterIdx, &bm.CharOffset, &bm.Anchor, &bm.ScrollPct, &bm.Note, &bm.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &bm, nil
}

// DeleteBookmark removes a bookmark owned by userID.
// Authorization check is inlined in the WHERE clause.
func DeleteBookmark(ctx context.Context, userID, bookmarkID int) error {
	result, err := database.Pool.Exec(ctx,
		`DELETE FROM bookmarks WHERE id = $1 AND user_id = $2`,
		bookmarkID, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateBookmarkNote updates only the note field of a bookmark.
// Position (chapter_idx / char_offset) is intentionally immutable — if the
// user wants to move a bookmark they should delete and re-create.
func UpdateBookmarkNote(ctx context.Context, userID, bookmarkID int, note string) (*model.Bookmark, error) {
	var bm model.Bookmark
	err := database.Pool.QueryRow(ctx,
		`UPDATE bookmarks SET note = $1
		 WHERE id = $2 AND user_id = $3
		 RETURNING id, user_id, book_id, chapter_idx, char_offset, anchor, scroll_pct, note, created_at`,
		note, bookmarkID, userID,
	).Scan(&bm.ID, &bm.UserID, &bm.BookID, &bm.ChapterIdx, &bm.CharOffset, &bm.Anchor, &bm.ScrollPct, &bm.Note, &bm.CreatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &bm, nil
}
