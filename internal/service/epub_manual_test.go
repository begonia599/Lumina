package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEPUBRealBook(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "epub", "不正经的魔术讲师与禁忌教典（1-24）.epub")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read epub fixture: %v", err)
	}

	result, err := ParseEPUB(data)
	if err != nil {
		t.Fatalf("ParseEPUB failed: %v", err)
	}

	if result.IsFixedLayout {
		t.Fatal("expected reflowable EPUB")
	}
	if result.Title == "" {
		t.Fatal("expected title")
	}
	if len(result.Chapters) == 0 {
		t.Fatal("expected chapters")
	}
	if result.PlainText == "" {
		t.Fatal("expected extracted plain text")
	}

	t.Logf("title=%q author=%q chapters=%d plain=%d cover=%t first=%q last=%q",
		result.Title,
		result.Author,
		len(result.Chapters),
		len(result.PlainText),
		len(result.CoverData) > 0,
		result.Chapters[0].Title,
		result.Chapters[len(result.Chapters)-1].Title,
	)
}
