package service

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// DetectEncoding detects the character encoding of the given byte slice.
// Returns the detected encoding name (e.g. "UTF-8", "GB-18030").
func DetectEncoding(data []byte) string {
	// Fast path: if it's valid UTF-8, just return
	if utf8.Valid(data) {
		return "UTF-8"
	}

	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(data)
	if err != nil || result == nil {
		return "UTF-8" // fallback
	}

	return result.Charset
}

// ConvertToUTF8 converts the byte data from the detected encoding to UTF-8.
func ConvertToUTF8(data []byte, charset string) ([]byte, error) {
	if charset == "UTF-8" || charset == "utf-8" {
		return data, nil
	}

	enc := lookupEncoding(charset)
	if enc == nil {
		return nil, fmt.Errorf("unsupported encoding: %s", charset)
	}

	reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to convert from %s to UTF-8: %w", charset, err)
	}

	return result, nil
}

// DetectAndConvert detects encoding and converts to UTF-8 in one step.
// Returns the UTF-8 content and the original detected encoding name.
func DetectAndConvert(data []byte) ([]byte, string, error) {
	charset := DetectEncoding(data)
	converted, err := ConvertToUTF8(data, charset)
	if err != nil {
		return nil, charset, err
	}
	return converted, charset, nil
}

// lookupEncoding maps charset name strings to Go encoding objects.
func lookupEncoding(charset string) encoding.Encoding {
	switch charset {
	case "GB-18030", "GB18030", "gb18030":
		return simplifiedchinese.GB18030
	case "GBK", "gbk":
		return simplifiedchinese.GBK
	case "GB2312", "gb2312", "GB-2312":
		return simplifiedchinese.HZGB2312
	case "Big5", "big5", "Big-5", "BIG5":
		return traditionalchinese.Big5
	case "EUC-KR", "euc-kr":
		return korean.EUCKR
	case "EUC-JP", "euc-jp":
		return japanese.EUCJP
	case "Shift_JIS", "shift_jis", "SHIFT_JIS":
		return japanese.ShiftJIS
	case "ISO-2022-JP":
		return japanese.ISO2022JP
	case "UTF-16BE":
		return unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	case "UTF-16LE":
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	default:
		// Try GBK as the best fallback for Chinese text
		return simplifiedchinese.GBK
	}
}
