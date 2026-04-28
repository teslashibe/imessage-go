package imessage

import (
	"encoding/binary"
	"unicode/utf8"
)

// decodeAttributedBody extracts the primary text from an Apple
// NSAttributedString stored in chat.db's message.attributedBody column.
//
// The full format (Apple's typedstream / NSArchiver) is documented at
// https://chrissardegna.com/blog/reverse-engineering-apple-typedstream-format/
// — for our needs (the message body) we only need the first NSString
// instance after the class table. NSString primitives are encoded with
// the type tag '+' (0x2B) followed by a length prefix and UTF-8 bytes:
//
//	0x01 0x2B <len:1>            -> short string (len < 0x81)
//	0x01 0x2B 0x81 <len:le2>      -> medium string
//	0x01 0x2B 0x82 <len:le4>      -> long string
//
// We additionally accept a bare 0x2B prefix (no leading 0x01) which
// appears in some macOS versions, and validate the result is UTF-8.
//
// Returns "" when no plausible NSString is found; callers should fall
// back to the message.text column.
func decodeAttributedBody(b []byte) string {
	if len(b) < 4 {
		return ""
	}
	// Skip past the typedstream header magic if present so we don't match
	// inside class names.
	start := 0
	if i := indexBytes(b, []byte("NSString")); i >= 0 {
		// Move start past the last NSString class declaration, so we look
		// at *instance* data, not the class name itself.
		start = i + len("NSString")
	}
	for i := start; i < len(b)-2; i++ {
		// Look for the NSString primitive tag.
		var off int
		switch {
		case b[i] == 0x01 && b[i+1] == 0x2B:
			off = i + 2
		case b[i] == 0x2B && i > 0 && (b[i-1] == 0x84 || b[i-1] == 0x01 || b[i-1] == 0x86):
			off = i + 1
		default:
			continue
		}
		text, ok := readNSStringAt(b, off)
		if ok && text != "" {
			return text
		}
	}
	return ""
}

// readNSStringAt parses a length-prefixed UTF-8 string at offset off,
// returning (text, ok). Length encodings: <0x81 = single byte; 0x81 = 2-byte
// LE; 0x82 = 4-byte LE.
func readNSStringAt(b []byte, off int) (string, bool) {
	if off >= len(b) {
		return "", false
	}
	var length int
	switch {
	case b[off] == 0x81:
		if off+3 > len(b) {
			return "", false
		}
		length = int(binary.LittleEndian.Uint16(b[off+1 : off+3]))
		off += 3
	case b[off] == 0x82:
		if off+5 > len(b) {
			return "", false
		}
		length = int(binary.LittleEndian.Uint32(b[off+1 : off+5]))
		off += 5
	case b[off] < 0x81:
		length = int(b[off])
		off++
	default:
		return "", false
	}
	if length <= 0 || off+length > len(b) {
		return "", false
	}
	candidate := b[off : off+length]
	if !utf8.Valid(candidate) {
		return "", false
	}
	// Sanity check: a real message body shouldn't start with the typed-
	// stream class-name garbage. Reject candidates dominated by control
	// bytes.
	if controlRatio(candidate) > 0.3 {
		return "", false
	}
	return string(candidate), true
}

func controlRatio(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	ctl := 0
	for _, c := range b {
		if c < 0x09 || (c > 0x0D && c < 0x20) {
			ctl++
		}
	}
	return float64(ctl) / float64(len(b))
}

func indexBytes(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	if len(needle) > len(haystack) {
		return -1
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
