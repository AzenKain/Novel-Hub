package defaultcover

import (
	"fmt"
	"hash/fnv"
	"html"
	"strings"
)

var coverGradients = [][2]string{
	{"#1e1b4b", "#312e81"},
	{"#064e3b", "#047857"},
	{"#4c1d95", "#6d28d9"},
	{"#701a75", "#a21caf"},
	{"#1e293b", "#334155"},
	{"#831843", "#be185d"},
}

func wrapText(text string, maxLen int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var currentLine string
	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= maxLen {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

func GenerateSVG(title, author string) []byte {
	if title == "" {
		title = "Untitled Book"
	}
	if author == "" {
		author = "Unknown Author"
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(title + author))
	colorIdx := int(h.Sum32()) % len(coverGradients)
	bgGrad := coverGradients[colorIdx]

	titleLines := wrapText(title, 24)
	if len(titleLines) > 4 {
		titleLines = titleLines[:4]
		titleLines[3] += "..."
	}

	authorLines := wrapText(author, 38)
	if len(authorLines) > 2 {
		authorLines = authorLines[:2]
		authorLines[1] += "..."
	}

	titleLineHeight := 48
	titleStartY := 420 - (len(titleLines)-1)*titleLineHeight/2
	titleSVG := ""
	for i, line := range titleLines {
		y := titleStartY + i*titleLineHeight
		titleSVG += fmt.Sprintf(`  <text x="300" y="%d" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="36" font-weight="bold" fill="#ffffff" text-anchor="middle" dominant-baseline="middle">%s</text>
`, y, html.EscapeString(line))
	}

	authorStartY := titleStartY + len(titleLines)*titleLineHeight + 40
	authorLineHeight := 30
	authorSVG := ""
	for i, line := range authorLines {
		y := authorStartY + i*authorLineHeight
		authorSVG += fmt.Sprintf(`  <text x="300" y="%d" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="22" font-weight="normal" fill="#e2e8f0" text-anchor="middle" dominant-baseline="middle">%s</text>
`, y, html.EscapeString(line))
	}

	svg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="600" height="900" viewBox="0 0 600 900" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="bg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" stop-color="%s"/>
      <stop offset="100%%" stop-color="%s"/>
    </linearGradient>
  </defs>
  <rect width="600" height="900" fill="url(#bg)"/>
  <rect x="30" y="30" width="540" height="840" rx="16" fill="none" stroke="#ffffff" stroke-opacity="0.15" stroke-width="2"/>
%s%s  <text x="300" y="820" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="14" font-weight="600" fill="#94a3b8" letter-spacing="4" text-anchor="middle">NOVELHUB DIGITAL</text>
</svg>`, bgGrad[0], bgGrad[1], titleSVG, authorSVG)

	return []byte(svg)
}
