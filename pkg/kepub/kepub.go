package kepub

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
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

func ConvertEPUBToKePub(epubReader io.ReaderAt, size int64, out io.Writer) (err error) {
	zr, err := zip.NewReader(epubReader, size)
	if err != nil {
		return fmt.Errorf("failed to open epub zip reader: %w", err)
	}
	if len(zr.File) > constants.MaxArchiveEntries {
		return fmt.Errorf("epub archive has too many entries")
	}

	var total uint64
	for _, file := range zr.File {
		if file.UncompressedSize64 > constants.MaxArchiveAssetSize {
			return fmt.Errorf("epub entry %s exceeds size limit", file.Name)
		}
		if file.UncompressedSize64 > uint64(constants.MaxArchiveUncompressedBytes)-total {
			return fmt.Errorf("epub archive exceeds uncompressed size limit")
		}
		total += file.UncompressedSize64
	}

	zw := zip.NewWriter(out)
	defer func() {
		if closeErr := zw.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("failed to close kepub zip writer: %w", closeErr)
		}
	}()

	var actualTotal int64
	for _, file := range zr.File {
		rc, openErr := file.Open()
		if openErr != nil {
			return fmt.Errorf("failed to open file %s in epub: %w", file.Name, openErr)
		}

		fh := file.FileHeader
		fw, createErr := zw.CreateHeader(&fh)
		if createErr != nil {
			_ = rc.Close()
			return fmt.Errorf("failed to create zip header for %s: %w", file.Name, createErr)
		}

		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext == ".xhtml" || ext == ".html" || ext == ".htm" {
			buf, readErr := bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
			closeErr := rc.Close()
			if readErr != nil {
				return fmt.Errorf("failed to read file %s: %w", file.Name, readErr)
			}
			if closeErr != nil {
				return fmt.Errorf("failed to close file %s: %w", file.Name, closeErr)
			}
			actualTotal += int64(len(buf))
			if actualTotal > constants.MaxArchiveUncompressedBytes {
				return fmt.Errorf("epub archive exceeds uncompressed size limit")
			}
			if _, err = io.WriteString(fw, InjectKoboSpans(string(buf))); err != nil {
				return fmt.Errorf("failed to write zip file content for %s: %w", file.Name, err)
			}
			continue
		}

		written, copyErr := io.Copy(fw, io.LimitReader(rc, constants.MaxArchiveAssetSize+1))
		closeErr := rc.Close()
		if copyErr != nil {
			return fmt.Errorf("failed to copy file %s: %w", file.Name, copyErr)
		}
		if written > constants.MaxArchiveAssetSize {
			return fmt.Errorf("epub entry %s exceeds size limit", file.Name)
		}
		actualTotal += written
		if actualTotal > constants.MaxArchiveUncompressedBytes {
			return fmt.Errorf("epub archive exceeds uncompressed size limit")
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close file %s: %w", file.Name, closeErr)
		}
	}

	return nil
}
