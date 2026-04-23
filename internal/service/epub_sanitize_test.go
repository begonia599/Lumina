package service

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"
)

func TestSanitizeStylesheetRealBook(t *testing.T) {
	zr := openRealBookZip(t)
	cssBytes, err := readZipEntry(zr, "OPS/css/main.css")
	if err != nil {
		t.Fatalf("read main.css: %v", err)
	}

	css := sanitizeStylesheet(string(cssBytes), 42, "OPS/css", "#ch-1")
	if css == "" {
		t.Fatal("expected sanitized css")
	}
	if !strings.Contains(css, "#ch-1") {
		t.Fatalf("expected scoped selectors, got: %s", css[:min(len(css), 300)])
	}
	if strings.Contains(strings.ToLower(css), "res:///") {
		t.Fatal("expected proprietary res:// URLs to be removed")
	}
	if !strings.Contains(css, `font-family: "cnepub"`) && !strings.Contains(css, `font-family:"cnepub"`) {
		t.Fatal("expected safe declarations to survive sanitization")
	}
}

func TestRewriteHrefCrossChapter(t *testing.T) {
	chapterRefs := map[string]int{
		"OPS/chapter2.html": 1,
		"OPS/chapter3.html": 2,
	}

	got, external := rewriteHref("chapter3.html#section-2", 1, "OPS", chapterRefs)
	if external {
		t.Fatal("expected internal chapter jump")
	}
	want := "#/chapter/2#ch-2-section-2"
	if got != want {
		t.Fatalf("rewriteHref mismatch: got %q want %q", got, want)
	}
}

func TestSanitizeInlineStyleBlocksDangerousDeclarations(t *testing.T) {
	style := `position: fixed; color: red; background-image: url("https://evil.invalid/x.png"); width: 10em;`
	got := sanitizeInlineStyle(style, 7, "OPS", "ch-3-")

	if strings.Contains(strings.ToLower(got), "fixed") {
		t.Fatalf("dangerous position should be removed: %q", got)
	}
	if strings.Contains(strings.ToLower(got), "evil.invalid") {
		t.Fatalf("external url should be removed: %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "color: red") {
		t.Fatalf("safe declaration should remain: %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "width: 10em") {
		t.Fatalf("safe width should remain: %q", got)
	}
}

func TestExtractChapterCSSRealBook(t *testing.T) {
	zr := openRealBookZip(t)
	chapterBytes, err := readZipEntry(zr, "OPS/chapter2.html")
	if err != nil {
		t.Fatalf("read chapter2.html: %v", err)
	}

	doc, err := xhtml.Parse(bytes.NewReader(chapterBytes))
	if err != nil {
		t.Fatalf("parse chapter html: %v", err)
	}

	css := extractChapterCSS(zr, doc, 42, 1, "OPS/chapter2.html")
	if css == "" {
		t.Fatal("expected chapter css")
	}
	if !strings.Contains(css, "#ch-1 p") {
		t.Fatalf("expected scoped p selector, got: %s", css[:min(len(css), 300)])
	}
	if strings.Contains(strings.ToLower(css), "res:///") {
		t.Fatal("expected proprietary res URL to be stripped")
	}
}

func TestSanitizeHTMLNodeBlocksDangerousContentAndRewritesResources(t *testing.T) {
	input := `<div>
		<script>alert(1)</script>
		<p id="note" onclick="evil()" style="position: fixed; color: red; width: 12em;">Hello</p>
		<a href="javascript:alert(1)">bad</a>
		<a href="chapter3.html#sec">next</a>
		<img src="../images/pic.jpg" onerror="evil()" />
	</div>`
	doc, err := xhtml.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse input html: %v", err)
	}

	nodes := sanitizeChildren(findBodyNode(doc), 9, 1, "OPS/chapter2.html", map[string]int{
		"OPS/chapter2.html": 1,
		"OPS/chapter3.html": 2,
	})
	html := renderNodes(t, nodes)

	if strings.Contains(strings.ToLower(html), "<script") {
		t.Fatalf("script should be removed: %s", html)
	}
	if strings.Contains(strings.ToLower(html), "onclick") || strings.Contains(strings.ToLower(html), "onerror") {
		t.Fatalf("event handlers should be removed: %s", html)
	}
	if strings.Contains(strings.ToLower(html), "javascript:") {
		t.Fatalf("javascript href should be removed: %s", html)
	}
	if strings.Contains(strings.ToLower(html), "position: fixed") {
		t.Fatalf("dangerous inline style should be removed: %s", html)
	}
	if !strings.Contains(html, `id="ch-1-note"`) {
		t.Fatalf("id should be prefixed: %s", html)
	}
	if !strings.Contains(html, `style="color: red; width: 12em"`) && !strings.Contains(html, `style="color: red; width: 12em;"`) {
		t.Fatalf("safe inline style should remain: %s", html)
	}
	if !strings.Contains(html, `href="#/chapter/2#ch-2-sec"`) {
		t.Fatalf("cross chapter href should be rewritten: %s", html)
	}
	if !strings.Contains(html, `/api/books/9/resources/images/pic.jpg`) {
		t.Fatalf("resource URL should be rewritten: %s", html)
	}
}

func TestRewriteResourceURLBlocksUnsafeSchemes(t *testing.T) {
	cases := []string{
		"javascript:alert(1)",
		"https://evil.invalid/a.png",
		"data:image/png;base64,abc",
		"#local-anchor",
	}

	for _, input := range cases {
		if got := rewriteResourceURL(3, "OPS", input); got != "" {
			t.Fatalf("expected %q to be rejected, got %q", input, got)
		}
	}
}

func openRealBookZip(t *testing.T) *zip.Reader {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "epub", "不正经的魔术讲师与禁忌教典（1-24）.epub")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read epub fixture: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open epub zip: %v", err)
	}
	return zr
}

func renderNodes(t *testing.T, nodes []*xhtml.Node) string {
	t.Helper()

	var out bytes.Buffer
	for _, node := range nodes {
		if err := xhtml.Render(&out, node); err != nil {
			t.Fatalf("render node: %v", err)
		}
	}
	return out.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
