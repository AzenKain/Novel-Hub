package docx

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
)

func parseTable(decoder *xml.Decoder, relMap map[string]string) (*tableElement, error) {
	table := &tableElement{}
	var currentRow *tableRow
	var currentCell *tableCell
	var cellText strings.Builder
	var cellStyle runStyle
	inCellRun := false

	flushCellRun := func() {
		if inCellRun && cellText.Len() > 0 {
			if currentCell != nil {
				currentCell.elements = append(currentCell.elements, &textElement{
					text:  cellText.String(),
					style: cellStyle,
				})
			}
			cellText.Reset()
		}
	}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode docx table: %w", err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tr":
				currentRow = &tableRow{}
			case "tc":
				currentCell = &tableCell{colSpan: 1, rowSpan: 1}
				cellText.Reset()
				cellStyle = runStyle{}
			case "gridSpan":
				if currentCell != nil {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							if v, err := strconv.Atoi(attr.Value); err == nil && v > 1 {
								currentCell.colSpan = v
							}
						}
					}
				}
			case "vMerge":
				if currentCell != nil {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" && attr.Value == "restart" {
							currentCell.rowSpan = 2
						}
					}
				}
			case "r":
				inCellRun = true
				cellStyle = runStyle{}
				cellText.Reset()
			case "b":
				if inCellRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						cellStyle.bold = true
					}
				}
			case "i":
				if inCellRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						cellStyle.italic = true
					}
				}
			case "u":
				if inCellRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "none" && val != "0" {
						cellStyle.underline = true
					}
				}
			case "strike":
				if inCellRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						cellStyle.strike = true
					}
				}
			case "t":
				var text string
				if err := decoder.DecodeElement(&text, &t); err != nil {
					return nil, fmt.Errorf("decode docx table text: %w", err)
				}
				if inCellRun {
					cellText.WriteString(text)
				}
			case "br", "cr":
				if inCellRun {
					cellText.WriteByte('\n')
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "r":
				flushCellRun()
				inCellRun = false
			case "tc":
				flushCellRun()
				if currentCell != nil && currentRow != nil {
					currentRow.cells = append(currentRow.cells, *currentCell)
					currentCell = nil
				}
			case "tr":
				if currentRow != nil {
					table.rows = append(table.rows, *currentRow)
					currentRow = nil
				}
			case "tbl":
				return table, nil
			}
		}
	}
	return table, nil
}

func renderTable(table *tableElement) string {
	if table == nil || len(table.rows) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(`<div class="novelhub-table-wrapper"><table class="novelhub-table">`)
	for _, row := range table.rows {
		out.WriteString("<tr>")
		for _, cell := range row.cells {
			out.WriteString("<td")
			if cell.colSpan > 1 {
				fmt.Fprintf(&out, ` colspan="%d"`, cell.colSpan)
			}
			if cell.rowSpan > 1 {
				fmt.Fprintf(&out, ` rowspan="%d"`, cell.rowSpan)
			}
			out.WriteString(">")
			for _, el := range cell.elements {
				if te, ok := el.(*textElement); ok {
					escaped := html.EscapeString(te.text)
					if te.style.bold {
						escaped = "<b>" + escaped + "</b>"
					}
					if te.style.italic {
						escaped = "<i>" + escaped + "</i>"
					}
					out.WriteString(escaped)
				}
			}
			out.WriteString("</td>")
		}
		out.WriteString("</tr>")
	}
	out.WriteString("</table></div>")
	return out.String()
}
