package doc

import (
	"html"
	"strings"
)

func formatsToHTML(pieces []textPiece, formats []charFormat, paraFormats []paraFormat) string {
	type textSegment struct {
		text      string
		bold      bool
		italic    bool
		underline uint8
		strike    bool
		super     bool
		sub       bool
	}

	segments := make([]textSegment, 0, len(pieces))
	for _, p := range pieces {
		segments = append(segments, textSegment{text: p.text})
	}

	type paraRun struct {
		paraStart int
		paraEnd   int
	}
	var paraRuns []paraRun
	tableStart := -1
	for i, pf := range paraFormats {
		if pf.inTable {
			if tableStart == -1 {
				tableStart = i
			}
			continue
		}
		if tableStart != -1 {
			paraRuns = append(paraRuns, paraRun{tableStart, i})
			tableStart = -1
		}
	}
	if tableStart != -1 {
		paraRuns = append(paraRuns, paraRun{tableStart, len(paraFormats)})
	}

	var out strings.Builder
	out.WriteString("<article>")

	itemIdx := 0
	for _, run := range paraRuns {
		out.WriteString(`<div class="novelhub-table-wrapper"><table class="novelhub-table">`)

		for rowIdx := run.paraStart; rowIdx < run.paraEnd; rowIdx++ {
			out.WriteString("<tr>")
			raw := segments[rowIdx].text
			parts := strings.Split(raw, "\x07")
			for _, cell := range parts {
				cell = strings.TrimSpace(cell)
				if cell == "" {
					continue
				}
				if rowIdx == run.paraStart {
					out.WriteString("<td><p>")
				} else {
					out.WriteString("<td><p>")
				}
				safe := html.EscapeString(cell)
				safe = strings.ReplaceAll(safe, "\n", "<br>")
				out.WriteString(safe)
				out.WriteString("</p></td>")
			}
			out.WriteString("</tr>")
			itemIdx++
		}
		out.WriteString("</table></div>")
	}

	pos := 0
	for i := range segments {
		startPos := pos
		segLen := len([]rune(segments[i].text))
		if segLen == 0 {
			continue
		}
		text := segments[i].text
		runes := []rune(text)
		hasFormatting := false
		if startPos < len(formats) {
			f := formats[startPos]
			if f.bold || f.italic || f.underline != 0 || f.strike || f.superScript || f.subScript {
				hasFormatting = true
			}
		}
		if hasFormatting {
			f := formats[startPos]
			if f.superScript {
				out.WriteString("<sup>")
			} else if f.subScript {
				out.WriteString("<sub>")
			}
			if f.bold {
				out.WriteString("<b>")
			}
			if f.italic {
				out.WriteString("<i>")
			}
			if f.underline != 0 {
				out.WriteString("<u>")
			}
			if f.strike {
				out.WriteString("<s>")
			}
			for ri, r := range runes {
				_ = ri
				out.WriteString(html.EscapeString(string(r)))
			}
			if f.strike {
				out.WriteString("</s>")
			}
			if f.underline != 0 {
				out.WriteString("</u>")
			}
			if f.italic {
				out.WriteString("</i>")
			}
			if f.bold {
				out.WriteString("</b>")
			}
			if f.superScript {
				out.WriteString("</sup>")
			} else if f.subScript {
				out.WriteString("</sub>")
			}
		} else {
			out.WriteString(html.EscapeString(text))
		}
		pos += segLen
	}
	out.WriteString("</article>")
	return out.String()
}

func writeFormatTransitions(prev, cur charFormat, out *strings.Builder) {
	if prev.superScript {
		out.WriteString("</sup>")
	} else if prev.subScript {
		out.WriteString("</sub>")
	}
	if prev.strike {
		out.WriteString("</s>")
	}
	if prev.underline != 0 {
		out.WriteString("</u>")
	}
	if prev.italic {
		out.WriteString("</i>")
	}
	if prev.bold {
		out.WriteString("</b>")
	}
	if cur.bold {
		out.WriteString("<b>")
	}
	if cur.italic {
		out.WriteString("<i>")
	}
	if cur.underline != 0 {
		out.WriteString("<u>")
	}
	if cur.strike {
		out.WriteString("<s>")
	}
	if cur.superScript {
		out.WriteString("<sup>")
	} else if cur.subScript {
		out.WriteString("<sub>")
	}
}

func closeAllFormatting(f charFormat, out *strings.Builder) {
	if f.superScript {
		out.WriteString("</sup>")
	} else if f.subScript {
		out.WriteString("</sub>")
	}
	if f.strike {
		out.WriteString("</s>")
	}
	if f.underline != 0 {
		out.WriteString("</u>")
	}
	if f.italic {
		out.WriteString("</i>")
	}
	if f.bold {
		out.WriteString("</b>")
	}
}
