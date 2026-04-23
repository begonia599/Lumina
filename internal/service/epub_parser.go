package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"path"
	"strings"

	xhtml "golang.org/x/net/html"
)

type EPUBResult struct {
	Title         string
	Author        string
	Description   string
	Language      string
	Chapters      []EPUBChapter
	CoverData     []byte
	CoverMIME     string
	PlainText     string
	IsFixedLayout bool
}

type EPUBChapter struct {
	Title      string
	ContentRef string
	StartPos   int
	EndPos     int
}

type epubContainer struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type epubPackage struct {
	Metadata epubMetadata `xml:"metadata"`
	Manifest struct {
		Items []epubManifestItem `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		TOC      string `xml:"toc,attr"`
		Itemrefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
	Guide struct {
		References []struct {
			Type string `xml:"type,attr"`
			Href string `xml:"href,attr"`
		} `xml:"reference"`
	} `xml:"guide"`
}

type epubMetadata struct {
	Titles       []string       `xml:"title"`
	Creators     []string       `xml:"creator"`
	Descriptions []string       `xml:"description"`
	Languages    []string       `xml:"language"`
	Metas        []epubMetaItem `xml:"meta"`
}

type epubMetaItem struct {
	Name     string `xml:"name,attr"`
	Content  string `xml:"content,attr"`
	Property string `xml:"property,attr"`
	Value    string `xml:",chardata"`
}

type epubManifestItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type ncxDoc struct {
	NavMap struct {
		Points []ncxPoint `xml:"navPoint"`
	} `xml:"navMap"`
}

type ncxPoint struct {
	NavLabel struct {
		Text string `xml:"text"`
	} `xml:"navLabel"`
	Content struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	Children []ncxPoint `xml:"navPoint"`
}

func ParseEPUB(data []byte) (*EPUBResult, error) {
	zr, err := newEPUBReader(data)
	if err != nil {
		return nil, err
	}

	opfPath, err := findOPFPath(zr)
	if err != nil {
		return nil, err
	}
	opfBytes, err := readZipEntry(zr, opfPath)
	if err != nil {
		return nil, err
	}

	var pkg epubPackage
	if err := xml.Unmarshal(opfBytes, &pkg); err != nil {
		return nil, fmt.Errorf("parse opf: %w", err)
	}

	manifest := make(map[string]epubManifestItem, len(pkg.Manifest.Items))
	for _, item := range pkg.Manifest.Items {
		manifest[item.ID] = item
	}

	result := &EPUBResult{
		Title:       firstNonEmpty(pkg.Metadata.Titles...),
		Author:      firstNonEmpty(pkg.Metadata.Creators...),
		Description: firstNonEmpty(pkg.Metadata.Descriptions...),
		Language:    firstNonEmpty(pkg.Metadata.Languages...),
	}

	for _, meta := range pkg.Metadata.Metas {
		if strings.EqualFold(strings.TrimSpace(meta.Property), "rendition:layout") &&
			strings.EqualFold(strings.TrimSpace(meta.Value), "pre-paginated") {
			result.IsFixedLayout = true
			break
		}
	}

	opfDir := path.Dir(opfPath)
	tocTitles := buildTOCTitleMap(zr, opfDir, pkg, manifest)
	coverPath := findCoverPath(opfDir, pkg, manifest)
	if coverPath != "" {
		if coverData, err := readZipEntry(zr, coverPath); err == nil {
			result.CoverData = coverData
			result.CoverMIME = detectEPUBMime(coverPath, coverData)
		}
	}

	var plain strings.Builder
	for idx, itemref := range pkg.Spine.Itemrefs {
		item, ok := manifest[itemref.IDRef]
		if !ok {
			continue
		}

		contentRef := resolveEPUBPath(opfDir, item.Href)
		chapterBytes, err := readZipEntry(zr, contentRef)
		if err != nil {
			return nil, fmt.Errorf("read chapter %q: %w", contentRef, err)
		}

		chapterText := extractVisibleText(chapterBytes)
		if plain.Len() > 0 && chapterText != "" {
			plain.WriteString("\n\n")
		}
		start := plain.Len()
		plain.WriteString(chapterText)
		end := plain.Len()

		title := strings.TrimSpace(tocTitles[stripFragment(contentRef)])
		if title == "" {
			title = extractDocumentTitle(chapterBytes)
		}
		if title == "" {
			title = fmt.Sprintf("第 %d 章", idx+1)
		}

		result.Chapters = append(result.Chapters, EPUBChapter{
			Title:      title,
			ContentRef: contentRef,
			StartPos:   start,
			EndPos:     end,
		})
	}

	if result.Title == "" {
		result.Title = "Untitled EPUB"
	}
	result.PlainText = plain.String()
	return result, nil
}

func newEPUBReader(data []byte) (*zip.Reader, error) {
	readerAt := bytes.NewReader(data)
	zr, err := zip.NewReader(readerAt, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open epub zip: %w", err)
	}
	return zr, nil
}

func findOPFPath(zr *zip.Reader) (string, error) {
	data, err := readZipEntry(zr, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("read container.xml: %w", err)
	}
	var container epubContainer
	if err := xml.Unmarshal(data, &container); err != nil {
		return "", fmt.Errorf("parse container.xml: %w", err)
	}
	if len(container.Rootfiles) == 0 || strings.TrimSpace(container.Rootfiles[0].FullPath) == "" {
		return "", fmt.Errorf("epub missing rootfile")
	}
	return normalizeEPUBPath(container.Rootfiles[0].FullPath), nil
}

func readZipEntry(zr *zip.Reader, name string) ([]byte, error) {
	name = normalizeEPUBPath(name)
	for _, f := range zr.File {
		if normalizeEPUBPath(f.Name) != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("zip entry not found: %s", name)
}

func normalizeEPUBPath(p string) string {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	p = strings.TrimPrefix(path.Clean("/"+p), "/")
	if p == "." {
		return ""
	}
	return p
}

func resolveEPUBPath(baseDir, href string) string {
	href = stripFragment(strings.TrimSpace(href))
	if href == "" {
		return normalizeEPUBPath(baseDir)
	}
	return normalizeEPUBPath(path.Join(baseDir, href))
}

func stripFragment(href string) string {
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		return href[:idx]
	}
	return href
}

func buildTOCTitleMap(zr *zip.Reader, opfDir string, pkg epubPackage, manifest map[string]epubManifestItem) map[string]string {
	if navPath := findNavPath(opfDir, manifest); navPath != "" {
		if data, err := readZipEntry(zr, navPath); err == nil {
			return parseNavTitles(navPath, data)
		}
	}
	if pkg.Spine.TOC != "" {
		if item, ok := manifest[pkg.Spine.TOC]; ok {
			if data, err := readZipEntry(zr, resolveEPUBPath(opfDir, item.Href)); err == nil {
				return parseNCXTitles(opfDir, item.Href, data)
			}
		}
	}
	for _, item := range manifest {
		if item.MediaType == "application/x-dtbncx+xml" {
			if data, err := readZipEntry(zr, resolveEPUBPath(opfDir, item.Href)); err == nil {
				return parseNCXTitles(opfDir, item.Href, data)
			}
		}
	}
	return map[string]string{}
}

func findNavPath(opfDir string, manifest map[string]epubManifestItem) string {
	for _, item := range manifest {
		props := strings.Fields(strings.ToLower(item.Properties))
		for _, prop := range props {
			if prop == "nav" {
				return resolveEPUBPath(opfDir, item.Href)
			}
		}
	}
	return ""
}

func parseNavTitles(navPath string, data []byte) map[string]string {
	doc, err := xhtml.Parse(bytes.NewReader(data))
	if err != nil {
		return map[string]string{}
	}

	result := map[string]string{}
	var visit func(*xhtml.Node, bool)
	visit = func(n *xhtml.Node, inTOC bool) {
		if n.Type == xhtml.ElementNode && n.Data == "nav" {
			nextInTOC := inTOC || isTOCNav(n)
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				visit(child, nextInTOC)
			}
			return
		}
		if inTOC && n.Type == xhtml.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			title := strings.TrimSpace(extractNodeText(n))
			if href != "" && title != "" {
				resolved := resolveEPUBPath(path.Dir(navPath), href)
				result[stripFragment(resolved)] = title
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child, inTOC)
		}
	}
	visit(doc, false)
	return result
}

func isTOCNav(n *xhtml.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == "epub:type" || attr.Key == "type" || attr.Key == "role" {
			if strings.Contains(strings.ToLower(attr.Val), "toc") {
				return true
			}
		}
	}
	return false
}

func parseNCXTitles(opfDir, href string, data []byte) map[string]string {
	var doc ncxDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return map[string]string{}
	}
	baseDir := path.Dir(resolveEPUBPath(opfDir, href))
	result := map[string]string{}
	var walk func([]ncxPoint)
	walk = func(points []ncxPoint) {
		for _, point := range points {
			target := resolveEPUBPath(baseDir, point.Content.Src)
			title := strings.TrimSpace(point.NavLabel.Text)
			if target != "" && title != "" {
				result[stripFragment(target)] = title
			}
			walk(point.Children)
		}
	}
	walk(doc.NavMap.Points)
	return result
}

func findCoverPath(opfDir string, pkg epubPackage, manifest map[string]epubManifestItem) string {
	for _, meta := range pkg.Metadata.Metas {
		if strings.EqualFold(strings.TrimSpace(meta.Name), "cover") {
			if item, ok := manifest[strings.TrimSpace(meta.Content)]; ok {
				return resolveEPUBPath(opfDir, item.Href)
			}
		}
	}
	for _, item := range manifest {
		props := strings.Fields(strings.ToLower(item.Properties))
		for _, prop := range props {
			if prop == "cover-image" {
				return resolveEPUBPath(opfDir, item.Href)
			}
		}
	}
	for _, ref := range pkg.Guide.References {
		if strings.EqualFold(strings.TrimSpace(ref.Type), "cover") {
			return resolveEPUBPath(opfDir, ref.Href)
		}
	}
	return ""
}

func detectEPUBMime(name string, data []byte) string {
	extType := mime.TypeByExtension(strings.ToLower(path.Ext(name)))
	if extType != "" {
		return extType
	}
	if strings.HasPrefix(string(bytes.TrimSpace(data)), "<svg") {
		return "image/svg+xml"
	}
	return "application/octet-stream"
}

func extractDocumentTitle(data []byte) string {
	doc, err := xhtml.Parse(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	var title string
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if title != "" {
			return
		}
		if n.Type == xhtml.ElementNode && (n.Data == "title" || n.Data == "h1" || n.Data == "h2" || n.Data == "h3") {
			if text := strings.TrimSpace(extractNodeText(n)); text != "" {
				title = text
				return
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return title
}

func extractVisibleText(data []byte) string {
	doc, err := xhtml.Parse(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	var b strings.Builder
	var walk func(*xhtml.Node, bool)
	walk = func(n *xhtml.Node, skip bool) {
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "script", "style", "head":
				skip = true
			case "p", "div", "section", "article", "header", "footer", "aside", "blockquote",
				"li", "tr", "h1", "h2", "h3", "h4", "h5", "h6", "br", "hr":
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
			}
		}
		if !skip && n.Type == xhtml.TextNode {
			text := normalizeText(n.Data)
			if text != "" {
				if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
					b.WriteByte(' ')
				}
				b.WriteString(text)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child, skip)
		}
	}
	walk(doc, false)
	return strings.TrimSpace(compactBlankLines(b.String()))
}

func normalizeText(s string) string {
	return strings.Join(strings.Fields(cleanDisplayText(s)), " ")
}

func compactBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	lastBlank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if lastBlank {
				continue
			}
			lastBlank = true
			out = append(out, "")
			continue
		}
		lastBlank = false
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func extractNodeText(n *xhtml.Node) string {
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.TextNode {
			b.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return cleanDisplayText(b.String())
}

func getAttr(n *xhtml.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(cleanDisplayText(value))
		if value != "" {
			return value
		}
	}
	return ""
}
