package auth

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Username validation rules (ADR-11):
//   - 1–64 Unicode code points after trimming
//   - Any script allowed (Chinese, Japanese, emoji, symbols, punctuation)
//   - Rejected: control chars, NULL, zero-width chars (ZWSP/ZWJ/ZWNJ),
//     bidirectional overrides (LRO/RLO/PDF/LRE/RLE), pure whitespace
//
// SQLi is NOT prevented by character whitelisting — it is prevented by
// parameterized queries (ADR-12). Therefore special SQL-looking characters
// ARE permitted in usernames.
const (
	UsernameMinLen = 1
	UsernameMaxLen = 64
	PasswordMinLen = 8
	PasswordMaxLen = 256 // bcrypt truncates at 72 bytes; we cap earlier for sanity
)

// forbiddenUsernameRunes lists specific code points we refuse regardless of
// Unicode category (zero-width, bidi overrides, format controls commonly
// used for homograph / spoofing attacks).
var forbiddenUsernameRunes = map[rune]struct{}{
	0x0000: {}, // NULL
	0x200B: {}, // ZERO WIDTH SPACE
	0x200C: {}, // ZERO WIDTH NON-JOINER
	0x200D: {}, // ZERO WIDTH JOINER
	0x200E: {}, // LEFT-TO-RIGHT MARK
	0x200F: {}, // RIGHT-TO-LEFT MARK
	0x202A: {}, // LRE
	0x202B: {}, // RLE
	0x202C: {}, // PDF
	0x202D: {}, // LRO
	0x202E: {}, // RLO
	0x2066: {}, // LRI
	0x2067: {}, // RLI
	0x2068: {}, // FSI
	0x2069: {}, // PDI
	0xFEFF: {}, // BOM / ZWNBSP
}

// NormalizeUsername applies NFC + trim and returns the canonical form.
// It does NOT validate; call ValidateUsername next.
func NormalizeUsername(raw string) string {
	n := norm.NFC.String(raw)
	return strings.TrimSpace(n)
}

// ValidateUsername returns nil if the (already-normalized) name is acceptable.
func ValidateUsername(name string) error {
	if !utf8.ValidString(name) {
		return ErrInvalidUsername
	}
	runeCount := utf8.RuneCountInString(name)
	if runeCount < UsernameMinLen || runeCount > UsernameMaxLen {
		return ErrInvalidUsername
	}
	// Reject if any rune is a control char or in the forbidden set.
	for _, r := range name {
		if _, bad := forbiddenUsernameRunes[r]; bad {
			return ErrInvalidUsername
		}
		// Control chars (Cc) — tabs, line breaks etc. — rejected.
		if unicode.IsControl(r) {
			return ErrInvalidUsername
		}
	}
	// Reject pure-whitespace (after NFC+trim this is "", but double-check).
	if strings.TrimSpace(name) == "" {
		return ErrInvalidUsername
	}
	return nil
}

// ValidatePassword applies minimal length bounds.
// Password character set is entirely unrestricted (bcrypt handles bytes).
func ValidatePassword(pw string) error {
	if len(pw) < PasswordMinLen || len(pw) > PasswordMaxLen {
		return ErrInvalidPassword
	}
	return nil
}
