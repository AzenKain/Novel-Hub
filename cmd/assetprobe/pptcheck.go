package main

import (
	"fmt"
	"os"

	"novelhub/pkg/bookparser/presentation"
)

func pptCheck(path string) {
	data, _ := os.ReadFile(path)
	_ = data
	p := presentation.NewParser()
	spine, err := p.ParseSpine(path)
	if err != nil {
		fmt.Println("spine error:", err)
		return
	}
	fmt.Printf("chapters: %d\n", len(spine))
	for i, ch := range spine {
		content, err := p.GetChapterContent(path, ch.ContentPath)
		if err != nil {
			fmt.Printf("  %s err=%v\n", ch.ContentPath, err)
			continue
		}
		if i < 3 {
			snippet := content
			if len(snippet) > 120 {
				snippet = snippet[:120]
			}
			fmt.Printf("  [%d] %q: %s...\n", i, ch.Title, snippet)
		} else {
			fmt.Printf("  [%d] %q (%d chars)\n", i, ch.Title, len(content))
		}
	}
}
