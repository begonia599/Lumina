package model

// Position is a format-agnostic reading location.
// TXT uses ChapterIdx + CharOffset; EPUB uses ChapterIdx + Anchor/ScrollPct.
type Position struct {
	ChapterIdx int      `json:"chapterIdx"`
	CharOffset int      `json:"charOffset,omitempty"`
	Anchor     *string  `json:"anchor,omitempty"`
	ScrollPct  *float64 `json:"scrollPct,omitempty"`
}
