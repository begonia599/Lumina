package model

import "time"

// ReadingProgress is the per-user, per-book read location.
// The authoritative position is (ChapterIdx, CharOffset) — see ADR-3.
// Percentage is a derived field, persisted only for quick UI display.
type ReadingProgress struct {
	ID         int       `json:"id"`
	UserID     int       `json:"-"`
	BookID     int       `json:"bookId"`
	ChapterIdx int       `json:"chapterIdx"`
	CharOffset int       `json:"charOffset"`
	Anchor     *string   `json:"anchor,omitempty"`
	ScrollPct  *float64  `json:"scrollPct,omitempty"`
	Percentage float64   `json:"percentage"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
