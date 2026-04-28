package imessage

import "testing"

// Synthetic attributedBody payload that mimics the wire format closely
// enough to exercise the decoder: a "NSString" class-name marker followed
// by a short NSString instance ("hello world") prefixed by 0x01 0x2B.
func TestDecodeAttributedBody_ShortString(t *testing.T) {
	payload := []byte{
		// fake header
		0x04, 0x0b, 's', 't', 'r', 'e', 'a', 'm', 't', 'y', 'p', 'e', 'd',
		// class name
		0x08, 'N', 'S', 'S', 't', 'r', 'i', 'n', 'g',
		// instance: 0x01 0x2B <len=11> "hello world"
		0x01, 0x2B, 0x0b,
		'h', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd',
	}
	got := decodeAttributedBody(payload)
	if got != "hello world" {
		t.Fatalf("decodeAttributedBody = %q, want %q", got, "hello world")
	}
}

func TestDecodeAttributedBody_Empty(t *testing.T) {
	if got := decodeAttributedBody(nil); got != "" {
		t.Fatalf("empty bytes -> %q, want empty", got)
	}
	if got := decodeAttributedBody([]byte{0x01, 0x02}); got != "" {
		t.Fatalf("garbage bytes -> %q, want empty", got)
	}
}

func TestMarshalForJS_EscapesParaSeparators(t *testing.T) {
	// U+2028 and U+2029 are valid JSON string bytes but terminate JS
	// string literals. marshalForJS must escape them so the embedded
	// payload remains a valid JS expression.
	in := map[string]string{"body": "hello\u2028world\u2029!"}
	out, err := marshalForJS(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"\u2028", "\u2029"} {
		if contains := indexBytes([]byte(out), []byte(banned)); contains >= 0 {
			t.Fatalf("marshalForJS leaked raw separator at %d in %q", contains, out)
		}
	}
	for _, want := range []string{`\u2028`, `\u2029`} {
		if contains := indexBytes([]byte(out), []byte(want)); contains < 0 {
			t.Fatalf("marshalForJS missing escaped %q in %q", want, out)
		}
	}
}

func TestNormalizeHandleExported(t *testing.T) {
	// The package-level NormalizeHandle wrapper must agree with the
	// internal normalizer (drift would silently break the
	// imessage_check_imessage tool's "normalized" return field).
	if got, want := NormalizeHandle("+1 (415) 555-1212"), normalizeHandle("+1 (415) 555-1212"); got != want {
		t.Fatalf("NormalizeHandle = %q, internal = %q", got, want)
	}
}

func TestNormalizeHandle(t *testing.T) {
	cases := map[string]string{
		"+1 (415) 555-1212": "+14155551212",
		"4155551212":        "+14155551212",
		"14155551212":       "+14155551212",
		"  Alice@Example.COM  ": "alice@example.com",
		"":                  "",
	}
	for in, want := range cases {
		if got := normalizeHandle(in); got != want {
			t.Errorf("normalizeHandle(%q) = %q, want %q", in, got, want)
		}
	}
}
