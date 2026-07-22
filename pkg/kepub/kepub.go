package kepub

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

var tagRegex = regexp.MustCompile(`(?i)<(p|div|li|blockquote)[^>]*>`)

func InjectKoboSpans(html string) string {
	spanIndex := 1
	var result strings.Builder

	lastIdx := 0
	locs := tagRegex.FindAllStringIndex(html, -1)
	if len(locs) == 0 {
		return html
	}

	for _, loc := range locs {
		result.WriteString(html[lastIdx:loc[1]])
		spanTag := fmt.Sprintf(`<span class="koboSpan" id="koboSpan-%d">`, spanIndex)
		result.WriteString(spanTag)
		spanIndex++
		lastIdx = loc[1]
	}
	result.WriteString(html[lastIdx:])
	return result.String()
}

func splitSentences(text string) []string {
	var res []string
	var current strings.Builder
	runes := []rune(text)
	inTag := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		}

		current.WriteRune(r)

		if !inTag && (r == '.' || r == '!' || r == '?') {
			if i+1 == len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' || runes[i+1] == '<' {
				res = append(res, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		res = append(res, current.String())
	}
	return res
}

func ConvertEPUBToKePub(epubReader io.ReaderAt, size int64, out io.Writer) error {
	zr, err := zip.NewReader(epubReader, size)
	if err != nil {
		return fmt.Errorf("failed to open epub zip reader: %w", err)
	}

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, file := range zr.File {
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in epub: %w", file.Name, err)
		}

		buf, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file.Name, err)
		}

		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext == ".xhtml" || ext == ".html" || ext == ".htm" {
			modifiedHTML := InjectKoboSpans(string(buf))
			buf = []byte(modifiedHTML)
		}

		fh := file.FileHeader
		fh.UncompressedSize64 = uint64(len(buf))
		fw, err := zw.CreateHeader(&fh)
		if err != nil {
			return fmt.Errorf("failed to create zip header for %s: %w", file.Name, err)
		}
		if _, err := fw.Write(buf); err != nil {
			return fmt.Errorf("failed to write zip file content for %s: %w", file.Name, err)
		}
	}

	return zw.Close()
}
