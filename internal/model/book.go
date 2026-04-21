package model

import "time"

// Book is a per-user uploaded text file with its parsed chapter metadata.
type Book struct {
	ID           int       `json:"id"`
	UserID       int       `json:"-"`
	Title        string    `json:"title"`
	Author       string    `json:"author"`
	Description  string    `json:"description"`
	Tags         []string  `json:"tags"`
	Filename     string    `json:"filename"`
	FilePath     string    `json:"-"`
	CoverPath    string    `json:"-"`
	HasCover     bool      `json:"hasCover"`
	FileSize     int64     `json:"fileSize"`
	Encoding     string    `json:"encoding"`
	ChapterCount int       `json:"chapterCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// BookWithProgress is returned on the bookshelf listing.
type BookWithProgress struct {
	Book
	Progress *ReadingProgress `json:"progress,omitempty"`
}
