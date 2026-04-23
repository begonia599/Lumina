package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"lumina/internal/database"

	xhtml "golang.org/x/net/html"
)

var epubAllowedTags = map[string]bool{
	"a": true, "abbr": true, "article": true, "aside": true, "b": true, "bdi": true, "bdo": true,
	"blockquote": true, "br": true, "caption": true, "cite": true, "code": true, "col": true,
	"colgroup": true, "dd": true, "defs": true, "details": true, "dfn": true, "div": true,
	"dl": true, "dt": true, "em": true, "figcaption": true, "figure": true, "footer": true,
	"g": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"header": true, "hgroup": true, "hr": true, "i": true, "img": true, "kbd": true, "li": true,
	"line": true, "main": true, "mark": true, "nav": true, "ol": true, "p": true, "path": true,
	"picture": true, "polygon": true, "polyline": true, "pre": true, "q": true, "rect": true,
	"rp": true, "rt": true, "ruby": true, "s": true, "samp": true, "section": true, "small": true,
	"source": true, "span": true, "strong": true, "sub": true, "summary": true, "sup": true,
	"svg": true, "symbol": true, "table": true, "tbody": true, "td": true, "text": true,
	"tfoot": true, "th": true, "thead": true, "time": true, "tr": true, "tspan": true, "u": true,
	"ul": true, "var": true, "wbr": true,
}

var epubBlockedTags = map[string]bool{
	"button": true, "embed": true, "form": true, "head": true, "html": true, "iframe": true,
	"input": true, "link": true, "meta": true, "object": true, "script": true, "style": true,
	"textarea": true,
}

var epubAllowedAttrs = map[string]bool{
	"alt": true, "class": true, "colspan": true, "cx": true, "cy": true, "d": true, "datetime": true,
	"dir": true, "fill": true, "headers": true, "height": true, "href": true, "id": true, "lang": true,
	"points": true, "preserveaspectratio": true, "r": true, "role": true, "rowspan": true,
	"scope": true, "sizes": true, "src": true, "srcset": true, "stroke": true, "stroke-width": true,
	"style": true, "title": true, "transform": true, "viewbox": true, "width": true, "x": true, "x1": true,
	"x2": true, "xmlns": true, "xmlns:xlink": true, "y": true, "y1": true, "y2": true,
}

func SanitizeChapterHTML(ctx context.Context, zipReader *zip.Reader, bookID int, chapterIdx int, contentRef string) (string, string, error) {
	data, err := readZipEntry(zipReader, contentRef)
	if err != nil {
		return "", "", err
	}
	doc, err := xhtml.Parse(bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("parse chapter html: %w", err)
	}
	chapterRefs, err := loadChapterRefIndex(ctx, bookID)
	if err != nil {
		return "", "", err
	}
	css := extractChapterCSS(zipReader, doc, bookID, chapterIdx, contentRef)

	root := &xhtml.Node{
		Type: xhtml.ElementNode,
		Data: "div",
		Attr: []xhtml.Attribute{
			{Key: "class", Val: "epub-chapter"},
			{Key: "id", Val: fmt.Sprintf("ch-%d", chapterIdx)},
		},
	}

	sourceRoot := findBodyNode(doc)
	for child := sourceRoot.FirstChild; child != nil; child = child.NextSibling {
		for _, sanitized := range sanitizeHTMLNode(child, bookID, chapterIdx, contentRef, chapterRefs) {
			root.AppendChild(sanitized)
		}
	}

	var out bytes.Buffer
	if err := xhtml.Render(&out, root); err != nil {
		return "", "", fmt.Errorf("render sanitized html: %w", err)
	}
	return out.String(), css, nil
}

func findBodyNode(doc *xhtml.Node) *xhtml.Node {
	var walk func(*xhtml.Node) *xhtml.Node
	walk = func(n *xhtml.Node) *xhtml.Node {
		if n.Type == xhtml.ElementNode && n.Data == "body" {
			return n
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if found := walk(child); found != nil {
				return found
			}
		}
		return nil
	}
	if body := walk(doc); body != nil {
		return body
	}
	return doc
}

func sanitizeHTMLNode(n *xhtml.Node, bookID int, chapterIdx int, contentRef string, chapterRefs map[string]int) []*xhtml.Node {
	switch n.Type {
	case xhtml.TextNode:
		return []*xhtml.Node{{Type: xhtml.TextNode, Data: n.Data}}
	case xhtml.ElementNode:
		tag := strings.ToLower(n.Data)
		if epubBlockedTags[tag] {
			return nil
		}
		if tag == "body" || tag == "html" {
			return sanitizeChildren(n, bookID, chapterIdx, contentRef, chapterRefs)
		}
		if !epubAllowedTags[tag] {
			return sanitizeChildren(n, bookID, chapterIdx, contentRef, chapterRefs)
		}

		node := &xhtml.Node{Type: xhtml.ElementNode, Data: tag}
		for _, attr := range sanitizeAttrs(n, bookID, chapterIdx, contentRef, chapterRefs) {
			node.Attr = append(node.Attr, attr)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			for _, sanitized := range sanitizeHTMLNode(child, bookID, chapterIdx, contentRef, chapterRefs) {
				node.AppendChild(sanitized)
			}
		}
		return []*xhtml.Node{node}
	default:
		return nil
	}
}

func sanitizeChildren(n *xhtml.Node, bookID int, chapterIdx int, contentRef string, chapterRefs map[string]int) []*xhtml.Node {
	var out []*xhtml.Node
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		out = append(out, sanitizeHTMLNode(child, bookID, chapterIdx, contentRef, chapterRefs)...)
	}
	return out
}

func sanitizeAttrs(n *xhtml.Node, bookID int, chapterIdx int, contentRef string, chapterRefs map[string]int) []xhtml.Attribute {
	baseDir := path.Dir(contentRef)
	out := make([]xhtml.Attribute, 0, len(n.Attr))
	for _, attr := range n.Attr {
		key := strings.ToLower(attr.Key)
		if strings.HasPrefix(key, "on") {
			continue
		}
		if !epubAllowedAttrs[key] {
			continue
		}

		val := strings.TrimSpace(attr.Val)
		switch key {
		case "id":
			if val == "" {
				continue
			}
			val = prefixAnchorID(chapterIdx, val)
		case "href":
			rewritten, keepExtra := rewriteHref(val, chapterIdx, baseDir, chapterRefs)
			if rewritten == "" {
				continue
			}
			val = rewritten
			out = append(out, xhtml.Attribute{Key: key, Val: val})
			if keepExtra {
				out = append(out,
					xhtml.Attribute{Key: "target", Val: "_blank"},
					xhtml.Attribute{Key: "rel", Val: "noopener noreferrer"},
				)
			}
			continue
		case "src":
			val = rewriteResourceURL(bookID, baseDir, val)
			if val == "" {
				continue
			}
		case "srcset":
			val = rewriteSrcset(bookID, baseDir, val)
			if val == "" {
				continue
			}
		case "style":
			val = sanitizeInlineStyle(val, bookID, baseDir, fmt.Sprintf("ch-%d-", chapterIdx))
			if val == "" {
				continue
			}
		case "xlink:href":
			val = rewriteResourceURL(bookID, baseDir, val)
			if val == "" {
				continue
			}
		}

		out = append(out, xhtml.Attribute{Key: attr.Key, Namespace: attr.Namespace, Val: val})
	}
	return out
}

func prefixAnchorID(chapterIdx int, id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return "ch-" + strconv.Itoa(chapterIdx) + "-" + id
}

func rewriteHref(href string, chapterIdx int, baseDir string, chapterRefs map[string]int) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" {
		return "", false
	}
	lower := strings.ToLower(href)
	switch {
	case strings.HasPrefix(lower, "javascript:"),
		strings.HasPrefix(lower, "vbscript:"),
		strings.HasPrefix(lower, "data:"),
		strings.HasPrefix(href, "//"):
		return "", false
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return href, true
	case strings.HasPrefix(href, "#"):
		id := strings.TrimPrefix(href, "#")
		if id == "" {
			return "", false
		}
		return "#" + prefixAnchorID(chapterIdx, id), false
	default:
		if u, err := url.Parse(href); err == nil && u.Scheme != "" {
			return "", false
		}
		targetRef := resolveEPUBPath(baseDir, href)
		targetIdx, ok := chapterRefs[stripFragment(targetRef)]
		if !ok {
			return "", false
		}
		anchor := extractHrefFragment(href)
		if targetIdx == chapterIdx {
			if anchor == "" {
				return "#ch-" + strconv.Itoa(chapterIdx), false
			}
			return "#" + prefixAnchorID(targetIdx, anchor), false
		}
		target := "#/chapter/" + strconv.Itoa(targetIdx)
		if anchor != "" {
			target += "#" + prefixAnchorID(targetIdx, anchor)
		}
		return target, false
	}
}

func extractHrefFragment(href string) string {
	if idx := strings.IndexByte(href, '#'); idx >= 0 && idx+1 < len(href) {
		return strings.TrimSpace(href[idx+1:])
	}
	return ""
}

func loadChapterRefIndex(ctx context.Context, bookID int) (map[string]int, error) {
	rows, err := database.Pool.Query(ctx,
		`SELECT chapter_idx, content_ref
		 FROM chapters
		 WHERE book_id = $1 AND content_ref IS NOT NULL AND content_ref <> ''`,
		bookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refs := make(map[string]int)
	for rows.Next() {
		var (
			chapterIdx int
			contentRef string
		)
		if err := rows.Scan(&chapterIdx, &contentRef); err != nil {
			return nil, err
		}
		refs[normalizeEPUBPath(contentRef)] = chapterIdx
	}
	return refs, rows.Err()
}

func rewriteResourceURL(bookID int, baseDir, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "#") {
		return ""
	}
	if u, err := url.Parse(ref); err == nil && u.Scheme != "" {
		return ""
	}
	resolved := resolveEPUBPath(baseDir, ref)
	if resolved == "" {
		return ""
	}
	return fmt.Sprintf("/api/books/%d/resources/%s", bookID, resolved)
}

func rewriteSrcset(bookID int, baseDir, srcset string) string {
	parts := strings.Split(srcset, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		rewritten := rewriteResourceURL(bookID, baseDir, fields[0])
		if rewritten == "" {
			continue
		}
		if len(fields) > 1 {
			out = append(out, rewritten+" "+strings.Join(fields[1:], " "))
			continue
		}
		out = append(out, rewritten)
	}
	return strings.Join(out, ", ")
}

func extractChapterCSS(zipReader *zip.Reader, doc *xhtml.Node, bookID int, chapterIdx int, contentRef string) string {
	baseDir := path.Dir(contentRef)
	scopeRoot := fmt.Sprintf("#ch-%d", chapterIdx)
	styles := make([]string, 0, 4)
	seen := make(map[string]struct{})

	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			switch strings.ToLower(n.Data) {
			case "style":
				raw := strings.TrimSpace(extractNodeText(n))
				if raw != "" {
					if css := sanitizeStylesheet(raw, bookID, baseDir, scopeRoot); css != "" {
						styles = append(styles, css)
					}
				}
			case "link":
				if !isStylesheetLink(n) {
					break
				}
				href := getAttr(n, "href")
				if href == "" {
					break
				}
				if u, err := url.Parse(strings.TrimSpace(href)); err == nil && u.Scheme != "" {
					break
				}
				cssRef := resolveEPUBPath(baseDir, href)
				if cssRef == "" {
					break
				}
				if _, ok := seen[cssRef]; ok {
					break
				}
				seen[cssRef] = struct{}{}
				data, err := readZipEntry(zipReader, cssRef)
				if err != nil {
					break
				}
				if css := sanitizeStylesheet(string(data), bookID, path.Dir(cssRef), scopeRoot); css != "" {
					styles = append(styles, css)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return strings.Join(styles, "\n")
}

func isStylesheetLink(n *xhtml.Node) bool {
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == "rel" && strings.Contains(strings.ToLower(attr.Val), "stylesheet") {
			return true
		}
	}
	return false
}
