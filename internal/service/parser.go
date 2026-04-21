package service

import (
	"regexp"
	"strings"

	"lumina/internal/model"
)

// chapterPatterns are regexps matching common Chinese novel chapter formats.
var chapterPatterns = []*regexp.Regexp{
	// 第一章 xxx / 第1章 xxx / 第一百二十三章 xxx
	regexp.MustCompile(`(?m)^\s*(第[零一二三四五六七八九十百千万壹贰叁肆伍陆柒捌玖拾佰仟\d]+[章节回卷集部篇])\s*[：:\s]?\s*(.*)$`),
	// Chapter 1 / CHAPTER 1: Title
	regexp.MustCompile(`(?mi)^\s*(Chapter\s+\d+)\s*[：:\s]?\s*(.*)$`),
	// 序章 / 序言 / 楔子 / 引子 / 前言 / 后记 / 尾声 / 番外
	regexp.MustCompile(`(?m)^\s*(序章|序言|楔子|引子|前言|后记|尾声|番外\S*)\s*[：:\s]?\s*(.*)$`),
}

// ChapterInfo represents a parsed chapter position in the text.
type ChapterInfo struct {
	Title    string
	StartPos int
	EndPos   int
}

// ParseChapters analyzes the full text content and extracts chapter boundaries.
// Returns a list of chapters with their positions in the text.
func ParseChapters(content string) []ChapterInfo {
	var matches []chapterMatch

	for _, pattern := range chapterPatterns {
		locs := pattern.FindAllStringIndex(content, -1)
		subs := pattern.FindAllStringSubmatch(content, -1)
		for i, loc := range locs {
			title := strings.TrimSpace(subs[i][1])
			subtitle := strings.TrimSpace(subs[i][2])
			if subtitle != "" {
				title = title + " " + subtitle
			}
			matches = append(matches, chapterMatch{title: title, pos: loc[0]})
		}
	}

	if len(matches) == 0 {
		// No chapters detected — treat entire text as a single chapter
		return []ChapterInfo{
			{
				Title:    "全文",
				StartPos: 0,
				EndPos:   len(content),
			},
		}
	}

	// Sort matches by position
	sortMatches(matches)

	// Remove duplicates at same position
	matches = dedup(matches)

	// Build chapter list with boundaries
	chapters := make([]ChapterInfo, 0, len(matches))

	// If there's content before the first chapter, add a "前言" section
	if matches[0].pos > 0 {
		preface := strings.TrimSpace(content[:matches[0].pos])
		if len(preface) > 50 { // Only add if there's meaningful content
			chapters = append(chapters, ChapterInfo{
				Title:    "前言",
				StartPos: 0,
				EndPos:   matches[0].pos,
			})
		}
	}

	for i, m := range matches {
		endPos := len(content)
		if i+1 < len(matches) {
			endPos = matches[i+1].pos
		}
		chapters = append(chapters, ChapterInfo{
			Title:    m.title,
			StartPos: m.pos,
			EndPos:   endPos,
		})
	}

	return chapters
}

// BuildChapterModels converts parsed ChapterInfo to model.Chapter with book_id.
func BuildChapterModels(bookID int, infos []ChapterInfo) []model.Chapter {
	chapters := make([]model.Chapter, len(infos))
	for i, info := range infos {
		chapters[i] = model.Chapter{
			BookID:     bookID,
			ChapterIdx: i,
			Title:      info.Title,
			StartPos:   info.StartPos,
			EndPos:     info.EndPos,
		}
	}
	return chapters
}

// chapterMatch is a helper type for sorting chapter positions.
type chapterMatch struct {
	title string
	pos   int
}

func sortMatches(matches []chapterMatch) {
	for i := 1; i < len(matches); i++ {
		key := matches[i]
		j := i - 1
		for j >= 0 && matches[j].pos > key.pos {
			matches[j+1] = matches[j]
			j--
		}
		matches[j+1] = key
	}
}

func dedup(matches []chapterMatch) []chapterMatch {
	if len(matches) <= 1 {
		return matches
	}
	result := []chapterMatch{matches[0]}
	for i := 1; i < len(matches); i++ {
		if matches[i].pos != result[len(result)-1].pos {
			result = append(result, matches[i])
		}
	}
	return result
}
