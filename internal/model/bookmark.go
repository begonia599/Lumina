package model

import "time"

// Bookmark is a named position saved by a user.
// Position is (ChapterIdx, CharOffset); Note is an optional annotation.
type Bookmark struct {
	ID         int       `json:"id"`
	UserID     int       `json:"-"`
	BookID     int       `json:"bookId"`
	ChapterIdx int       `json:"chapterIdx"`
	CharOffset int       `json:"charOffset"`
	Anchor     *string   `json:"anchor,omitempty"`
	ScrollPct  *float64  `json:"scrollPct,omitempty"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"createdAt"`
}
