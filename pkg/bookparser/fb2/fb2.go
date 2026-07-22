package fb2

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"os"
	"strings"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/jsonx"
)

type Parser struct{}

type fictionBook struct {
	Description fb2Description `xml:"description"`
	Bodies      []fb2Body      `xml:"body"`
	Binaries    []fb2Binary    `xml:"binary"`
}

type fb2Description struct {
	TitleInfo fb2TitleInfo `xml:"title-info"`
}

type fb2TitleInfo struct {
	Genres     []string      `xml:"genre"`
	Authors    []fb2Author   `xml:"author"`
	BookTitle  string        `xml:"book-title"`
	Annotation fb2Annotation `xml:"annotation"`
	Lang       string        `xml:"lang"`
	Sequences  []fb2Sequence `xml:"sequence"`
	Coverpage  fb2Coverpage  `xml:"coverpage"`
}

type fb2Author struct {
	FirstName  string `xml:"first-name"`
	MiddleName string `xml:"middle-name"`
	LastName   string `xml:"last-name"`
	NickName   string `xml:"nickname"`
}

type fb2Annotation struct {
	Paragraphs []fb2Paragraph `xml:"p"`
}

type fb2Sequence struct {
	Name   string `xml:"name,attr"`
	Number string `xml:"number,attr"`
}

type fb2Coverpage struct {
	Images []fb2Image `xml:"image"`
}

type fb2Body struct {
	Name     string       `xml:"name,attr"`
	Title    fb2Title     `xml:"title"`
	Sections []fb2Section `xml:"section"`
}

type fb2Section struct {
	ID         string         `xml:"id,attr"`
	Title      fb2Title       `xml:"title"`
	Paragraphs []fb2Paragraph `xml:"p"`
	Subtitles  []fb2Paragraph `xml:"subtitle"`
	Images     []fb2Image     `xml:"image"`
	Sections   []fb2Section   `xml:"section"`
}

type fb2Title struct {
	Paragraphs []fb2Paragraph `xml:"p"`
}

type fb2Paragraph struct {
	ID       string `xml:"id,attr"`
	InnerXML string `xml:",innerxml"`
	Text     string `xml:",chardata"`
}

type fb2Image struct {
	Href string
}

type fb2Binary struct {
	ID          string `xml:"id,attr"`
	ContentType string `xml:"content-type,attr"`
	Data        string `xml:",chardata"`
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	book, err := readFB2(filePath)
	if err != nil {
		return nil, err
	}
	info := book.Description.TitleInfo
	meta := &bookparser.BookMetadata{
		Title:       clean(info.BookTitle),
		Author:      strings.Join(authorNames(info.Authors), ", "),
		Description: annotationText(info.Annotation),
		Language:    clean(info.Lang),
		Subjects:    cleanList(info.Genres),
	}
	if meta.Title == "" {
		meta.Title = bookparser.TitleFromPath(filePath)
	}
	if len(info.Sequences) > 0 {
		meta.Series = clean(info.Sequences[0].Name)
		meta.SeriesIndex = clean(info.Sequences[0].Number)
	}
	if coverID := firstCoverID(info.Coverpage); coverID != "" {
		if data, contentType, ok := findBinary(book, coverID); ok {
			meta.CoverData = data
			meta.CoverType = contentType
		}
	}
	metaJSON, _ := jsonx.MarshalString(map[string]any{
		"title":       meta.Title,
		"creator":     meta.Author,
		"description": meta.Description,
		"language":    meta.Language,
		"subject":     meta.Subjects,
		"series":      meta.Series,
		"seriesIndex": meta.SeriesIndex,
		"format":      "fb2",
	})
	meta.MetadataJSON = metaJSON
	return bookparser.MergeMetadataSidecar(filePath, meta), nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	book, err := readFB2(filePath)
	if err != nil {
		return nil, err
	}
	sections := topSections(book)
	chapters := make([]bookparser.ChapterData, 0, len(sections))
	for index, section := range sections {
		title := sectionTitle(section)
		if title == "" {
			title = fmt.Sprintf("Section %d", index+1)
		}
		chapters = append(chapters, bookparser.ChapterData{
			Title:       title,
			ContentPath: fmt.Sprintf("section:%d", index),
			Index:       index,
		})
	}
	if len(chapters) == 0 {
		chapters = append(chapters, bookparser.ChapterData{
			Title:       bookparser.TitleFromPath(filePath),
			ContentPath: "section:0",
			Index:       0,
		})
	}
	return chapters, nil
}

func (p *Parser) ParseBook(filePath string) (*bookparser.BookData, error) {
	meta, err := p.ParseMetadata(filePath)
	if err != nil {
		return nil, err
	}
	chapters, err := p.ParseSpine(filePath)
	if err != nil {
		return nil, err
	}
	for i := range chapters {
		content, err := p.GetChapterContent(filePath, chapters[i].ContentPath)
		if err == nil {
			chapters[i].Content = content
		}
	}
	return &bookparser.BookData{Metadata: *meta, Chapters: chapters}, nil
}

func (p *Parser) GetChapterContent(filePath, contentPath string) (string, error) {
	book, err := readFB2(filePath)
	if err != nil {
		return "", err
	}
	sections := topSections(book)
	index := 0
	if strings.HasPrefix(contentPath, "section:") {
		_, _ = fmt.Sscanf(contentPath, "section:%d", &index)
	}
	sectionIndexByID := buildFB2SectionIndexMap(sections)
	var out strings.Builder
	out.WriteString(`<article class="novelhub-fb2">`)
	if len(sections) == 0 {
		for _, body := range book.Bodies {
			writeTitle(&out, body.Title, 1)
			for _, section := range body.Sections {
				writeSection(&out, section, 2, 0, sectionIndexByID)
			}
		}
	} else if index >= 0 && index < len(sections) {
		writeSection(&out, sections[index], 1, index, sectionIndexByID)
	} else {
		return "", fmt.Errorf("fb2 section %d not found", index)
	}
	out.WriteString(`</article>`)
	return out.String(), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	book, err := readFB2(filePath)
	if err != nil {
		return nil, err
	}
	id := strings.TrimPrefix(assetPath, "images/")
	id = strings.TrimPrefix(id, "#")
	if data, _, ok := findBinary(book, id); ok {
		return data, nil
	}
	return nil, fmt.Errorf("fb2 asset %q not found", assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	book, err := readFB2(filePath)
	if err != nil {
		return nil, err
	}
	images := make([]string, 0, len(book.Binaries))
	for _, binary := range book.Binaries {
		if strings.HasPrefix(strings.ToLower(binary.ContentType), "image/") {
			images = append(images, "images/"+binary.ID)
		}
	}
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func readFB2(filePath string) (*fictionBook, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read fb2: %w", err)
	}
	var book fictionBook
	decoder := xml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&book); err != nil {
		return nil, fmt.Errorf("decode fb2: %w", err)
	}
	return &book, nil
}

func (img *fb2Image) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "href" {
			img.Href = strings.TrimPrefix(strings.TrimSpace(attr.Value), "#")
			break
		}
	}
	return decoder.Skip()
}

func topSections(book *fictionBook) []fb2Section {
	if book == nil {
		return nil
	}
	for _, body := range book.Bodies {
		if strings.EqualFold(body.Name, "notes") || strings.EqualFold(body.Name, "comments") {
			continue
		}
		if len(body.Sections) > 0 {
			return body.Sections
		}
	}
	return nil
}

func sectionTitle(section fb2Section) string {
	return clean(strings.Join(titleParagraphs(section.Title), " "))
}

func writeSection(out *strings.Builder, section fb2Section, level int, currentTopIndex int, sectionIndexByID map[string]int) {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	out.WriteString("<section")
	if section.ID != "" {
		out.WriteString(` id="`)
		out.WriteString(html.EscapeString(section.ID))
		out.WriteString(`"`)
	}
	out.WriteString(">")
	writeTitle(out, section.Title, level)
	for _, image := range section.Images {
		writeImage(out, image)
	}
	for _, subtitle := range section.Subtitles {
		out.WriteString("<h")
		out.WriteByte(byte('0' + min(level+1, 6)))
		out.WriteString(">")
		out.WriteString(inlineXMLToHTML(subtitle, currentTopIndex, sectionIndexByID))
		out.WriteString("</h")
		out.WriteByte(byte('0' + min(level+1, 6)))
		out.WriteString(">")
	}
	for _, paragraph := range section.Paragraphs {
		out.WriteString("<p")
		if paragraph.ID != "" {
			out.WriteString(` id="`)
			out.WriteString(html.EscapeString(paragraph.ID))
			out.WriteString(`"`)
		}
		out.WriteString(">")
		out.WriteString(inlineXMLToHTML(paragraph, currentTopIndex, sectionIndexByID))
		out.WriteString("</p>")
	}
	for _, child := range section.Sections {
		writeSection(out, child, level+1, currentTopIndex, sectionIndexByID)
	}
	out.WriteString("</section>")
}

func writeTitle(out *strings.Builder, title fb2Title, level int) {
	paragraphs := titleParagraphs(title)
	if len(paragraphs) == 0 {
		return
	}
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	out.WriteString("<h")
	out.WriteByte(byte('0' + level))
	out.WriteString(">")
	out.WriteString(html.EscapeString(strings.Join(paragraphs, " ")))
	out.WriteString("</h")
	out.WriteByte(byte('0' + level))
	out.WriteString(">")
}

func writeImage(out *strings.Builder, image fb2Image) {
	if image.Href == "" {
		return
	}
	out.WriteString(`<figure><img src="images/`)
	out.WriteString(html.EscapeString(image.Href))
	out.WriteString(`" loading="lazy" /></figure>`)
}

func inlineXMLToHTML(paragraph fb2Paragraph, currentTopIndex int, sectionIndexByID map[string]int) string {
	value := strings.TrimSpace(paragraph.InnerXML)
	if value == "" {
		return html.EscapeString(clean(paragraph.Text))
	}
	decoder := xml.NewDecoder(strings.NewReader("<root>" + value + "</root>"))
	var out strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return html.EscapeString(clean(paragraph.Text))
		}
		switch t := token.(type) {
		case xml.CharData:
			out.WriteString(html.EscapeString(string(t)))
		case xml.StartElement:
			switch t.Name.Local {
			case "strong":
				out.WriteString("<strong>")
			case "emphasis":
				out.WriteString("<em>")
			case "strikethrough":
				out.WriteString("<s>")
			case "sub":
				out.WriteString("<sub>")
			case "sup":
				out.WriteString("<sup>")
			case "code":
				out.WriteString("<code>")
			case "a":
				out.WriteString(`<a href="`)
				out.WriteString(html.EscapeString(fb2LinkHref(t.Attr, currentTopIndex, sectionIndexByID)))
				out.WriteString(`">`)
			case "image":
				var image fb2Image
				for _, attr := range t.Attr {
					if attr.Name.Local == "href" {
						image.Href = strings.TrimPrefix(attr.Value, "#")
					}
				}
				writeImage(&out, image)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "strong":
				out.WriteString("</strong>")
			case "emphasis":
				out.WriteString("</em>")
			case "strikethrough":
				out.WriteString("</s>")
			case "sub":
				out.WriteString("</sub>")
			case "sup":
				out.WriteString("</sup>")
			case "code":
				out.WriteString("</code>")
			case "a":
				out.WriteString("</a>")
			}
		}
	}
	return strings.TrimSpace(out.String())
}

func fb2LinkHref(attrs []xml.Attr, currentTopIndex int, sectionIndexByID map[string]int) string {
	for _, attr := range attrs {
		if attr.Name.Local != "href" {
			continue
		}
		value := strings.TrimSpace(attr.Value)
		if value == "" {
			return "#"
		}
		if strings.HasPrefix(value, "#") {
			targetID := strings.TrimPrefix(value, "#")
			if targetIndex, ok := sectionIndexByID[targetID]; ok {
				if targetIndex == currentTopIndex {
					return "#" + targetID
				}
				return fmt.Sprintf("section:%d#%s", targetIndex, targetID)
			}
			return "#" + targetID
		}
		return value
	}
	return "#"
}

func buildFB2SectionIndexMap(sections []fb2Section) map[string]int {
	indexByID := make(map[string]int)
	for idx, section := range sections {
		collectFB2SectionIDs(section, idx, indexByID)
	}
	return indexByID
}

func collectFB2SectionIDs(section fb2Section, topIndex int, indexByID map[string]int) {
	if section.ID != "" {
		indexByID[section.ID] = topIndex
	}
	for _, paragraph := range section.Paragraphs {
		if paragraph.ID != "" {
			indexByID[paragraph.ID] = topIndex
		}
	}
	for _, subtitle := range section.Subtitles {
		if subtitle.ID != "" {
			indexByID[subtitle.ID] = topIndex
		}
	}
	for _, child := range section.Sections {
		collectFB2SectionIDs(child, topIndex, indexByID)
	}
}

func titleParagraphs(title fb2Title) []string {
	values := make([]string, 0, len(title.Paragraphs))
	for _, paragraph := range title.Paragraphs {
		value := clean(stripXML(paragraph))
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func stripXML(paragraph fb2Paragraph) string {
	if strings.TrimSpace(paragraph.InnerXML) == "" {
		return paragraph.Text
	}
	decoder := xml.NewDecoder(strings.NewReader("<root>" + paragraph.InnerXML + "</root>"))
	var out strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return paragraph.Text
		}
		if text, ok := token.(xml.CharData); ok {
			out.WriteString(string(text))
		}
	}
	return out.String()
}

func authorNames(authors []fb2Author) []string {
	names := make([]string, 0, len(authors))
	for _, author := range authors {
		parts := []string{author.FirstName, author.MiddleName, author.LastName}
		name := clean(strings.Join(parts, " "))
		if name == "" {
			name = clean(author.NickName)
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func annotationText(annotation fb2Annotation) string {
	parts := make([]string, 0, len(annotation.Paragraphs))
	for _, paragraph := range annotation.Paragraphs {
		if value := clean(stripXML(paragraph)); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "\n\n")
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = clean(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func firstCoverID(cover fb2Coverpage) string {
	for _, image := range cover.Images {
		if image.Href != "" {
			return image.Href
		}
	}
	return ""
}

func findBinary(book *fictionBook, id string) ([]byte, string, bool) {
	id = strings.TrimPrefix(id, "#")
	for _, binary := range book.Binaries {
		if binary.ID != id {
			continue
		}
		raw := strings.Join(strings.Fields(binary.Data), "")
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, "", false
		}
		return data, binary.ContentType, true
	}
	return nil, "", false
}

func clean(value string) string {
	return bookparser.CleanChapterTitle(value)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
