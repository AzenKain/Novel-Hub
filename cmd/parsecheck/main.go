package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/archivebook"
	"novelhub/pkg/bookparser/comic"
	docparser "novelhub/pkg/bookparser/doc"
	"novelhub/pkg/bookparser/docx"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/bookparser/fb2"
	"novelhub/pkg/bookparser/htmlfile"
	"novelhub/pkg/bookparser/mobi"
	"novelhub/pkg/bookparser/odt"
	"novelhub/pkg/bookparser/pdf"
	"novelhub/pkg/bookparser/plain"
	"novelhub/pkg/bookparser/rtf"
)

type result struct {
	Path         string   `json:"path"`
	Format       string   `json:"format"`
	OK           bool     `json:"ok"`
	Title        string   `json:"title,omitempty"`
	Chapters     int      `json:"chapters,omitempty"`
	Images       int      `json:"images,omitempty"`
	EmptyContent int      `json:"emptyContent,omitempty"`
	RawFile      bool     `json:"rawFile,omitempty"`
	DurationMS   int64    `json:"durationMs"`
	Errors       []string `json:"errors,omitempty"`
}

func main() {
	registry := newParserRegistry()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)

	failed := 0
	for _, path := range os.Args[1:] {
		res := checkPath(registry, path)
		if !res.OK {
			failed++
		}
		_ = encoder.Encode(res)
	}
	if failed > 0 {
		os.Exit(1)
	}
}

func newParserRegistry() *bookparser.Registry {
	registry := bookparser.NewRegistry()
	registry.Register(epub.NewParser(), "epub", "kepub.epub")
	registry.Register(plain.NewParser(), "txt", "md", "markdown")
	registry.Register(htmlfile.NewParser(), "html", "htm")
	registry.Register(docx.NewParser(), "docx")
	registry.Register(docparser.NewParser(), "doc")
	registry.Register(odt.NewParser(), "odt")
	registry.Register(rtf.NewParser(), "rtf")
	registry.Register(fb2.NewParser(), "fb2")
	registry.Register(pdf.NewParser(), "pdf")
	registry.Register(mobi.NewParser(), "mobi", "azw", "azw3", "amz")
	registry.Register(archivebook.NewParser("zip"), "zip")
	registry.Register(archivebook.NewParser("fbz"), "fbz")
	registry.Register(comic.NewParser("cbz"), "cbz")
	registry.Register(comic.NewParser("cbr"), "cbr")
	registry.Register(comic.NewParser("cbt"), "cbt")
	registry.Register(comic.NewParser("cb7"), "cb7")
	return registry
}

func checkPath(registry *bookparser.Registry, path string) (res result) {
	start := time.Now()
	res = result{
		Path:   filepath.ToSlash(path),
		Format: bookparser.FormatFromPath(path),
		OK:     true,
	}
	defer func() {
		res.DurationMS = time.Since(start).Milliseconds()
	}()

	parser, ok := registry.ParserForPath(path)
	if !ok {
		res.addError(fmt.Sprintf("no parser for format %q", res.Format))
		return res
	}
	if _, err := os.Stat(path); err != nil {
		res.addError("stat: " + err.Error())
		return res
	}

	meta, err := parser.ParseMetadata(path)
	if err != nil {
		res.addError("metadata: " + err.Error())
	}
	if meta != nil {
		res.Title = strings.TrimSpace(meta.Title)
	}

	spine, err := parser.ParseSpine(path)
	if err != nil {
		res.addError("spine: " + err.Error())
		return res
	}
	res.Chapters = len(spine)
	if len(spine) == 0 {
		res.addError("spine is empty")
		return res
	}

	for _, chapter := range spine {
		if chapter.ContentPath == bookparser.RawFileContentPath {
			res.RawFile = true
			continue
		}
		content, err := parser.GetChapterContent(path, chapter.ContentPath)
		if err != nil {
			res.addError(fmt.Sprintf("chapter %d %q: %s", chapter.Index, chapter.ContentPath, err.Error()))
			continue
		}
		if strings.TrimSpace(content) == "" {
			res.EmptyContent++
		}
	}

	images, err := parser.ListImages(path)
	if err != nil {
		res.addError("list images: " + err.Error())
		return res
	}
	res.Images = len(images)
	if len(images) > 0 {
		if _, err := parser.GetAsset(path, images[0]); err != nil {
			res.addError("first image asset: " + err.Error())
		}
	}

	return res
}

func (r *result) addError(value string) {
	r.OK = false
	r.Errors = append(r.Errors, value)
}
