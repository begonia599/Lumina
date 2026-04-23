package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

type cssSanitizeConfig struct {
	bookID          int
	baseDir         string
	anchorPrefix    string
	scopeRoot       string
	scopeSelectors  bool
	inlineStyleMode bool
	fontFaceMode    bool
}

var (
	cssRootSelectorPattern = regexp.MustCompile(`(?i)(^|[\s>+~,(])(html|body|:root)`)
	cssDangerPattern       = regexp.MustCompile(`(?i)(expression\s*\(|javascript:|vbscript:)`)
)

var inlineStyleAllowedPrefixes = []string{
	"background", "border", "font", "list-style", "margin", "padding", "text-",
}

var inlineStyleAllowedProps = map[string]bool{
	"clear":          true,
	"color":          true,
	"display":        true,
	"float":          true,
	"height":         true,
	"letter-spacing": true,
	"line-height":    true,
	"table-layout":   true,
	"text-indent":    true,
	"vertical-align": true,
	"white-space":    true,
	"width":          true,
	"word-spacing":   true,
}

var fontFaceAllowedProps = map[string]bool{
	"font-display":            true,
	"font-family":             true,
	"font-feature-settings":   true,
	"font-stretch":            true,
	"font-style":              true,
	"font-variation-settings": true,
	"font-weight":             true,
	"size-adjust":             true,
	"src":                     true,
	"unicode-range":           true,
}

func sanitizeStylesheet(cssText string, bookID int, baseDir, scopeRoot string) string {
	anchorPrefix := ""
	if scopeRoot != "" {
		anchorPrefix = strings.TrimPrefix(scopeRoot, "#") + "-"
	}
	cfg := cssSanitizeConfig{
		bookID:         bookID,
		baseDir:        baseDir,
		anchorPrefix:   anchorPrefix,
		scopeRoot:      scopeRoot,
		scopeSelectors: scopeRoot != "",
	}
	return strings.TrimSpace(parseCSSRules(stripCSSComments(cssText), cfg))
}

func sanitizeInlineStyle(style string, bookID int, baseDir, anchorPrefix string) string {
	cfg := cssSanitizeConfig{
		bookID:          bookID,
		baseDir:         baseDir,
		anchorPrefix:    anchorPrefix,
		inlineStyleMode: true,
	}
	return sanitizeDeclarationBlock(style, cfg)
}

func stripCSSComments(cssText string) string {
	var out strings.Builder
	for i := 0; i < len(cssText); {
		if i+1 < len(cssText) && cssText[i] == '/' && cssText[i+1] == '*' {
			end := strings.Index(cssText[i+2:], "*/")
			if end == -1 {
				break
			}
			i += end + 4
			continue
		}
		out.WriteByte(cssText[i])
		i++
	}
	return out.String()
}

func parseCSSRules(cssText string, cfg cssSanitizeConfig) string {
	var out strings.Builder
	for i := 0; i < len(cssText); {
		i = skipCSSWhitespace(cssText, i)
		if i >= len(cssText) {
			break
		}

		headerEnd, delim := findCSSHeaderEnd(cssText, i)
		if headerEnd < 0 {
			break
		}
		header := strings.TrimSpace(cssText[i:headerEnd])
		if header == "" {
			i = headerEnd + 1
			continue
		}

		if delim == ';' {
			i = headerEnd + 1
			continue
		}

		closeIdx := findMatchingBrace(cssText, headerEnd)
		if closeIdx < 0 {
			break
		}
		body := cssText[headerEnd+1 : closeIdx]
		if rule := processCSSRule(header, body, cfg); rule != "" {
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
			out.WriteString(rule)
		}
		i = closeIdx + 1
	}
	return out.String()
}

func processCSSRule(header, body string, cfg cssSanitizeConfig) string {
	lowerHeader := strings.ToLower(strings.TrimSpace(header))

	switch {
	case strings.HasPrefix(lowerHeader, "@import"), strings.HasPrefix(lowerHeader, "@charset"), strings.HasPrefix(lowerHeader, "@namespace"):
		return ""
	case strings.HasPrefix(lowerHeader, "@media"), strings.HasPrefix(lowerHeader, "@supports"):
		inner := parseCSSRules(body, cfg)
		if inner == "" {
			return ""
		}
		return header + "{" + inner + "}"
	case strings.HasPrefix(lowerHeader, "@font-face"):
		fontCfg := cfg
		fontCfg.scopeSelectors = false
		fontCfg.scopeRoot = ""
		fontCfg.fontFaceMode = true
		decls := sanitizeDeclarationBlock(body, fontCfg)
		if decls == "" {
			return ""
		}
		return "@font-face{" + decls + "}"
	case strings.Contains(lowerHeader, "keyframes"):
		keyCfg := cfg
		keyCfg.scopeSelectors = false
		keyCfg.scopeRoot = ""
		inner := parseCSSRules(body, keyCfg)
		if inner == "" {
			return ""
		}
		return header + "{" + inner + "}"
	case strings.HasPrefix(lowerHeader, "@"):
		return ""
	default:
		selectors := strings.TrimSpace(header)
		if cfg.scopeSelectors {
			selectors = scopeSelectorList(selectors, cfg.scopeRoot)
		}
		if selectors == "" {
			return ""
		}
		decls := sanitizeDeclarationBlock(body, cfg)
		if decls == "" {
			return ""
		}
		return selectors + "{" + decls + "}"
	}
}

func sanitizeDeclarationBlock(block string, cfg cssSanitizeConfig) string {
	declarations := splitTopLevel(block, ';')
	clean := make([]string, 0, len(declarations))
	for _, decl := range declarations {
		prop, value, ok := splitCSSDeclaration(decl)
		if !ok {
			continue
		}
		propLower := strings.ToLower(strings.TrimSpace(prop))
		value = strings.TrimSpace(value)
		if !allowCSSProperty(propLower, cfg) {
			continue
		}
		if cssDangerPattern.MatchString(value) {
			continue
		}
		if propLower == "position" {
			lowerValue := strings.ToLower(value)
			if strings.Contains(lowerValue, "fixed") || strings.Contains(lowerValue, "sticky") {
				continue
			}
		}
		if propLower == "behavior" || propLower == "-moz-binding" {
			continue
		}
		rewritten, ok := rewriteCSSURLs(value, cfg.bookID, cfg.baseDir, cfg.anchorPrefix)
		if !ok {
			continue
		}
		rewritten = strings.TrimSpace(rewritten)
		if rewritten == "" {
			continue
		}
		clean = append(clean, fmt.Sprintf("%s: %s", propLower, rewritten))
	}
	return strings.Join(clean, "; ")
}

func allowCSSProperty(prop string, cfg cssSanitizeConfig) bool {
	if prop == "" {
		return false
	}
	if cfg.fontFaceMode {
		return fontFaceAllowedProps[prop]
	}
	if !cfg.inlineStyleMode {
		return true
	}
	if inlineStyleAllowedProps[prop] {
		return true
	}
	for _, prefix := range inlineStyleAllowedPrefixes {
		if strings.HasPrefix(prop, prefix) {
			return true
		}
	}
	return false
}

func rewriteCSSURLs(value string, bookID int, baseDir, anchorPrefix string) (string, bool) {
	lowerValue := strings.ToLower(value)
	if strings.Contains(lowerValue, "@import") {
		return "", false
	}

	var out strings.Builder
	for i := 0; i < len(value); {
		idx := strings.Index(strings.ToLower(value[i:]), "url(")
		if idx < 0 {
			out.WriteString(value[i:])
			break
		}
		start := i + idx
		out.WriteString(value[i:start])

		argStart := start + 4
		argEnd := findCSSFunctionEnd(value, argStart)
		if argEnd < 0 {
			return "", false
		}

		rawRef := strings.TrimSpace(value[argStart:argEnd])
		rawRef = strings.Trim(rawRef, `"'`)
		if rawRef == "" {
			return "", false
		}
		if strings.HasPrefix(rawRef, "#") {
			if anchorPrefix == "" {
				return "", false
			}
			out.WriteString(`url("#`)
			out.WriteString(anchorPrefix)
			out.WriteString(strings.TrimPrefix(rawRef, "#"))
			out.WriteString(`")`)
			i = argEnd + 1
			continue
		}
		rewritten := rewriteResourceURL(bookID, baseDir, rawRef)
		if rewritten == "" {
			return "", false
		}
		out.WriteString(`url("`)
		out.WriteString(rewritten)
		out.WriteString(`")`)
		i = argEnd + 1
	}
	return out.String(), true
}

func scopeSelectorList(selectors, scopeRoot string) string {
	parts := splitTopLevel(selectors, ',')
	scoped := make([]string, 0, len(parts))
	for _, part := range parts {
		if sel := scopeSingleSelector(part, scopeRoot); sel != "" {
			scoped = append(scoped, sel)
		}
	}
	return strings.Join(scoped, ", ")
}

func scopeSingleSelector(selector, scopeRoot string) string {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return ""
	}

	scoped := cssRootSelectorPattern.ReplaceAllStringFunc(selector, func(match string) string {
		prefix := ""
		first := rune(match[0])
		if unicode.IsSpace(first) || strings.ContainsRune(">+~,(", first) {
			prefix = string(first)
		}
		return prefix + scopeRoot
	})

	if !strings.Contains(scoped, scopeRoot) {
		if strings.HasPrefix(scoped, "::") || strings.HasPrefix(scoped, ":") {
			scoped = scopeRoot + scoped
		} else {
			scoped = scopeRoot + " " + scoped
		}
	}
	double := scopeRoot + " " + scopeRoot
	for strings.Contains(scoped, double) {
		scoped = strings.ReplaceAll(scoped, double, scopeRoot)
	}
	return strings.TrimSpace(scoped)
}

func splitCSSDeclaration(decl string) (string, string, bool) {
	idx := findTopLevelColon(decl)
	if idx < 0 {
		return "", "", false
	}
	prop := strings.TrimSpace(decl[:idx])
	value := strings.TrimSpace(decl[idx+1:])
	if prop == "" || value == "" {
		return "", "", false
	}
	return prop, value, true
}

func splitTopLevel(input string, delim rune) []string {
	var (
		out      []string
		start    int
		paren    int
		bracket  int
		inQuote  rune
		escaping bool
	)

	for i, r := range input {
		if inQuote != 0 {
			if escaping {
				escaping = false
				continue
			}
			if r == '\\' {
				escaping = true
				continue
			}
			if r == inQuote {
				inQuote = 0
			}
			continue
		}

		switch r {
		case '\'', '"':
			inQuote = r
		case '(':
			paren++
		case ')':
			if paren > 0 {
				paren--
			}
		case '[':
			bracket++
		case ']':
			if bracket > 0 {
				bracket--
			}
		default:
			if r == delim && paren == 0 && bracket == 0 {
				out = append(out, input[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, input[start:])
	return out
}

func skipCSSWhitespace(input string, start int) int {
	for start < len(input) && unicode.IsSpace(rune(input[start])) {
		start++
	}
	return start
}

func findCSSHeaderEnd(input string, start int) (int, byte) {
	var (
		paren    int
		bracket  int
		inQuote  byte
		escaping bool
	)

	for i := start; i < len(input); i++ {
		ch := input[i]
		if inQuote != 0 {
			if escaping {
				escaping = false
				continue
			}
			if ch == '\\' {
				escaping = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inQuote = ch
		case '(':
			paren++
		case ')':
			if paren > 0 {
				paren--
			}
		case '[':
			bracket++
		case ']':
			if bracket > 0 {
				bracket--
			}
		case '{', ';':
			if paren == 0 && bracket == 0 {
				return i, ch
			}
		}
	}
	return -1, 0
}

func findMatchingBrace(input string, openIdx int) int {
	var (
		depth    = 1
		inQuote  byte
		escaping bool
	)

	for i := openIdx + 1; i < len(input); i++ {
		ch := input[i]
		if inQuote != 0 {
			if escaping {
				escaping = false
				continue
			}
			if ch == '\\' {
				escaping = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inQuote = ch
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findCSSFunctionEnd(input string, start int) int {
	var (
		depth    = 1
		inQuote  byte
		escaping bool
	)

	for i := start; i < len(input); i++ {
		ch := input[i]
		if inQuote != 0 {
			if escaping {
				escaping = false
				continue
			}
			if ch == '\\' {
				escaping = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inQuote = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findTopLevelColon(input string) int {
	var (
		paren    int
		bracket  int
		inQuote  rune
		escaping bool
	)

	for i, r := range input {
		if inQuote != 0 {
			if escaping {
				escaping = false
				continue
			}
			if r == '\\' {
				escaping = true
				continue
			}
			if r == inQuote {
				inQuote = 0
			}
			continue
		}

		switch r {
		case '\'', '"':
			inQuote = r
		case '(':
			paren++
		case ')':
			if paren > 0 {
				paren--
			}
		case '[':
			bracket++
		case ']':
			if bracket > 0 {
				bracket--
			}
		case ':':
			if paren == 0 && bracket == 0 {
				return i
			}
		}
	}
	return -1
}
