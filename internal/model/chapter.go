package model

type Chapter struct {
	ID         int    `json:"id"`
	BookID     int    `json:"bookId"`
	ChapterIdx int    `json:"chapterIdx"`
	Title      string `json:"title"`
	StartPos   int    `json:"startPos"`
	EndPos     int    `json:"endPos"`
}
