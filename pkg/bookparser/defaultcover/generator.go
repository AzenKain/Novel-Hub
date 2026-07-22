package defaultcover

import (
	"fmt"
	"hash/fnv"
	"html"
)

var coverGradients = [][2]string{
	{"#1e1b4b", "#312e81"},
	{"#064e3b", "#047857"},
	{"#4c1d95", "#6d28d9"},
	{"#701a75", "#a21caf"},
	{"#1e293b", "#334155"},
	{"#831843", "#be185d"},
}

// GenerateSVG produces a clean SVG cover image byte slice given title and author.
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

	safeTitle := html.EscapeString(title)
	safeAuthor := html.EscapeString(author)

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
  <text x="300" y="420" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="36" font-weight="bold" fill="#ffffff" text-anchor="middle" dominant-baseline="middle">%s</text>
  <text x="300" y="520" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="22" font-weight="normal" fill="#e2e8f0" text-anchor="middle" dominant-baseline="middle">%s</text>
  <text x="300" y="820" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="14" font-weight="600" fill="#94a3b8" letter-spacing="4" text-anchor="middle">NOVELHUB DIGITAL</text>
</svg>`, bgGrad[0], bgGrad[1], safeTitle, safeAuthor)

	return []byte(svg)
}
