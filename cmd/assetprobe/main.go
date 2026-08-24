package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	docparser "novelhub/pkg/bookparser/doc"
	"novelhub/pkg/bookparser/comic"
	"novelhub/pkg/bookparser/docx"
	"novelhub/pkg/bookparser/mobi"
	"novelhub/pkg/bookparser/odt"
	"novelhub/pkg/bookparser/presentation"
	"novelhub/pkg/bookparser/rtf"

	"novelhub/pkg/bookparser"
)

var imgRe = regexp.MustCompile(`<img\b[^>]*>`)

func dump(name, path string, p bookparser.Parser) {
	spine, err := p.ParseSpine(path)
	if err != nil {
		fmt.Println("ParseSpine error:", err)
		return
	}
	fmt.Printf("chapters: %d\n", len(spine))
	totalImgs := 0
	for i, ch := range spine {
		content, err := p.GetChapterContent(path, ch.ContentPath)
		if err != nil {
			fmt.Printf("  ch%d %q error: %v\n", i, ch.Title, err)
			continue
		}
		imgs := imgRe.FindAllString(content, -1)
		totalImgs += len(imgs)
		if len(imgs) > 0 || i < 2 {
			fmt.Printf("  ch%d %q (%d chars, %d imgs)\n", i, ch.Title, len(content), len(imgs))
			for j, im := range imgs {
				if j > 4 {
					break
				}
				fmt.Printf("    %s\n", im)
			}
			if i == 0 && len(imgs) == 0 {
			snippet := content
				if len(snippet) > 300 {
					snippet = snippet[:300]
				}
				fmt.Println("    head:", strings.ReplaceAll(snippet, "\n", " ")[:min(300, len(snippet))])
			}
		}
	}
	images, _ := p.ListImages(path)
	fmt.Printf("ListImages: %d -> %v\n", len(images), firstN(images, 5))
	ok, bad := 0, 0
	for _, img := range images {
		data, err := p.GetAsset(path, img)
		if err != nil || len(data) == 0 {
			bad++
			if bad <= 3 {
				fmt.Println("  GetAsset FAIL:", img, err)
			}
			continue
		}
		ok++
	}
	fmt.Printf("GetAsset ok=%d fail=%d\n", ok, bad)
}

func firstN(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	for _, arg := range os.Args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		name, path := parts[0], parts[1]
		fmt.Printf("\n########## %s ##########\n", name)
		switch {
		case strings.HasSuffix(path, ".mobi"):
			dump(name, path, mobi.NewParser())
		case strings.HasSuffix(path, ".rtf"):
			dump(name, path, rtf.NewParser())
		case strings.HasSuffix(path, ".odt"):
			dump(name, path, &odt.Parser{})
		case strings.HasSuffix(path, ".docx"):
			dump(name, path, docx.NewParser())
		case strings.HasSuffix(path, ".doc"):
			dump(name, path, &docparser.Parser{})
		case strings.HasSuffix(path, ".ppt") || strings.HasSuffix(path, ".pptx"):
			dump(name, path, presentation.NewParser())
			pptCheck(path)
		case strings.HasSuffix(path, ".cbz") || strings.HasSuffix(path, ".cb7") || strings.HasSuffix(path, ".cbt"):
			dump(name, path, comic.NewParser("cbz"))
		default:
			fmt.Println("unknown format")
		}
	}
}
