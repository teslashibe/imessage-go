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
