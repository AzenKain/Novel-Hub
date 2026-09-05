package ebookconv

import (
	"fmt"
	"strings"

	"novelhub/pkg/bookparser"
)

// SupportedTargets lists the output formats the pure-Go converters can write.
var SupportedTargets = []string{"epub", "fb2", "txt", "docx", "cbz", "kepub.epub", "mobi", "azw", "pdf"}

// Image is one embedded asset carried from the source into a target format.
type Image struct {
	Src  string
	Name string
	Data []byte
}

// Convert parses `path` via the registry and writes it as `target`.
func Convert(reg bookparser.Registry, format string, path string, target string) ([]byte, error) {
	target = bookparser.NormalizeFormat(target)
	if !IsTargetSupported(target) {
		return nil, fmt.Errorf("target format %q is not supported (supported: %s)", target, strings.Join(SupportedTargets, ", "))
	}
	if reg == nil {
		return nil, fmt.Errorf("parser registry is not configured")
	}
	parser, err := reg.Parser(format, path)
	if err != nil {
		return nil, err
	}
	if bookparser.NormalizeFormat(format) == target {
		return nil, fmt.Errorf("source and target format are the same (%q)", target)
	}
	book, err := parser.ParseBook(path)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", format, err)
	}
	images := collectImages(parser, path)
	switch target {
	case "epub":
		return writeEPUB(book, images)
	case "fb2":
		return writeFB2(book, images)
	case "txt":
		return writeTXT(book)
	case "docx":
		return writeDOCX(book, images)
	case "cbz":
		return writeCBZ(images)
	case "kepub.epub":
		return writeKEPUB(book, images)
	case "mobi", "azw":
		return writeMOBI(book, images)
	case "pdf":
		return writePDF(book, images)
	}
	return nil, fmt.Errorf("target format %q is not supported", target)
}

func IsTargetSupported(target string) bool {
	for _, t := range SupportedTargets {
		if t == bookparser.NormalizeFormat(target) {
			return true
		}
	}
	return false
}

func collectImages(parser bookparser.Parser, path string) []Image {
	names, err := parser.ListImages(path)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool, len(names))
	out := make([]Image, 0, len(names))
	for _, src := range names {
		if seen[src] {
			continue
		}
		seen[src] = true
		data, err := parser.GetAsset(path, src)
		if err != nil || len(data) == 0 {
			continue
		}
		out = append(out, Image{Src: src, Name: imageFilename(src, len(out)+1), Data: data})
	}
	return out
}
