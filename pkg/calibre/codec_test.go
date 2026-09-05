package calibre

import "testing"

func TestEncodeDecodeName(t *testing.T) {
	tests := []struct {
		raw     string
		encoded string
	}{
		{"authors", "617574686f7273"},
		{"series", "736572696573"},
		{"tags", "74616773"},
		{"Science Fiction", "536369656e63652046696374696f6e"},
		{"Tiếng Việt", "5469e1babf6e67205669e1bb8774"},
	}

	for _, tc := range tests {
		enc := EncodeName(tc.raw)
		if enc != tc.encoded {
			t.Errorf("EncodeName(%q) = %q, expected %q", tc.raw, enc, tc.encoded)
		}
		dec := DecodeName(enc)
		if dec != tc.raw {
			t.Errorf("DecodeName(%q) = %q, expected %q", enc, dec, tc.raw)
		}
	}

	if fallback := DecodeName("plain_text_name"); fallback != "plain_text_name" {
		t.Errorf("expected plain_text_name, got %q", fallback)
	}
	if empty := DecodeName(""); empty != "" {
		t.Errorf("expected empty, got %q", empty)
	}
}
