package kepub

import (
	"strings"
	"testing"
)

func TestInjectKoboSpans(t *testing.T) {
	inputHTML := `<html><body><p>Hello world. Welcome to NovelHub!</p><div>Another paragraph here.</div></body></html>`
	output := InjectKoboSpans(inputHTML)

	if !strings.Contains(output, `class="koboSpan"`) {
		t.Fatalf("expected koboSpan in output, got: %s", output)
	}
	if !strings.Contains(output, `id="koboSpan-1"`) {
		t.Fatalf("expected koboSpan-1 in output, got: %s", output)
	}
}
