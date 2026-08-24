package presentation

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
	"novelhub/pkg/constants"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pptx":
		return p.parsePPTXMetadata(filePath)
	case ".odp":
		return p.parseODPMetadata(filePath)
	case ".ppt":
		return p.parsePPTMetadata(filePath)
	default:
		return p.defaultMetadata(filePath), nil
	}
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pptx":
		return p.parsePPTXSpine(filePath)
	case ".odp":
		return p.parseODPSpine(filePath)
	case ".ppt":
		return p.parsePPTSpine(filePath)
	default:
		return p.defaultSpine(filePath), nil
	}
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
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pptx":
		return p.getPPTXSlideContent(filePath, contentPath)
	case ".odp":
		return p.getODPSlideContent(filePath, contentPath)
	case ".ppt":
		return p.getPPTContent(filePath, contentPath)
	default:
		return "<article><p>Unsupported presentation format.</p></article>", nil
	}
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	if assetPath == "" || !filepath.IsLocal(assetPath) {
		return nil, fmt.Errorf("invalid asset path")
	}

	if ext == ".ppt" {
		streams, err := readCompoundStreams(filePath)
		if err != nil {
			return nil, err
		}
		var assets []bookparser.EmbeddedImageAsset
		for _, data := range streams {
			assets = bookparser.AppendEmbeddedImageAssets(assets, data)
		}
		return bookparser.FindEmbeddedImageAsset(assets, assetPath)
	}

	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		cleanName := strings.TrimLeft(filepath.ToSlash(f.Name), "/")
		if cleanName == assetPath || strings.HasSuffix(cleanName, "/"+assetPath) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
		}
	}

	return nil, fmt.Errorf("asset %q not found", assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".ppt" {
		streams, err := readCompoundStreams(filePath)
		if err != nil {
			return []string{}, nil
		}
		var assets []bookparser.EmbeddedImageAsset
		for _, data := range streams {
			assets = bookparser.AppendEmbeddedImageAssets(assets, data)
		}
		return bookparser.EmbeddedImageNames(assets), nil
	}

	r, err := zip.OpenReader(filePath)
	if err != nil {
		return []string{}, nil
	}
	defer r.Close()

	var images []string
	for _, f := range r.File {
		lower := strings.ToLower(f.Name)
		if strings.HasPrefix(lower, "ppt/media/") || strings.HasPrefix(lower, "pictures/") || strings.HasPrefix(lower, "media/") || strings.HasPrefix(lower, "thumbnails/") {
			ext := filepath.Ext(lower)
			switch ext {
			case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".tiff",
				".mp3", ".m4a", ".wav", ".ogg", ".aac", ".wma", ".flac",
				".mp4", ".webm", ".m4v", ".ogv", ".avi", ".mov", ".wmv", ".mkv":
				images = append(images, f.Name)
			}
		}
	}
	sort.Strings(images)
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func (p *Parser) defaultMetadata(filePath string) *bookparser.BookMetadata {
	meta := &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}
	merged := bookparser.MergeMetadataSidecar(filePath, meta)
	if len(merged.CoverData) == 0 {
		merged.CoverData = defaultcover.GenerateSVG(merged.Title, merged.Author)
		merged.IsDefaultCover = true
		merged.CoverType = "image/svg+xml"
	}
	return merged
}

func (p *Parser) defaultSpine(filePath string) []bookparser.ChapterData {
	return []bookparser.ChapterData{{
		Title:       bookparser.TitleFromPath(filePath),
		ContentPath: "presentation",
		Index:       0,
	}}
}

// ----------------------------------------------------------------------------
// PPTX Support
// ----------------------------------------------------------------------------

func (p *Parser) parsePPTXMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return p.defaultMetadata(filePath), nil
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "docProps/core.xml" {
			rc, err := f.Open()
			if err == nil {
				var core struct {
					Title       string `xml:"title"`
					Creator     string `xml:"creator"`
					Description string `xml:"description"`
				}
				if xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&core) == nil {
					if t := strings.TrimSpace(core.Title); t != "" {
						meta.Title = t
					}
					meta.Author = strings.TrimSpace(core.Creator)
					meta.Description = strings.TrimSpace(core.Description)
				}
				rc.Close()
			}
			break
		}
	}

	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			if strings.HasSuffix(strings.ToLower(images[0]), ".png") {
				meta.CoverType = "image/png"
			} else {
				meta.CoverType = "image/jpeg"
			}
		}
	}

	merged := bookparser.MergeMetadataSidecar(filePath, meta)
	if len(merged.CoverData) == 0 {
		merged.CoverData = defaultcover.GenerateSVG(merged.Title, merged.Author)
		merged.IsDefaultCover = true
		merged.CoverType = "image/svg+xml"
	}
	return merged, nil
}

func (p *Parser) parsePPTXSpine(filePath string) ([]bookparser.ChapterData, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}
	defer r.Close()

	var slideFiles []string
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideFiles = append(slideFiles, f.Name)
		}
	}

	// Sort slides numerically: slide1.xml, slide2.xml, slide10.xml
	sort.Slice(slideFiles, func(i, j int) bool {
		numI := extractSlideNumber(slideFiles[i])
		numJ := extractSlideNumber(slideFiles[j])
		return numI < numJ
	})

	if len(slideFiles) == 0 {
		return p.defaultSpine(filePath), nil
	}

	var chapters []bookparser.ChapterData
	for i, slidePath := range slideFiles {
		title := fmt.Sprintf("Slide %d", i+1)
		slideTitle := p.extractPPTXSlideTitle(r, slidePath)
		if slideTitle != "" {
			title = fmt.Sprintf("Slide %d: %s", i+1, slideTitle)
		}
		chapters = append(chapters, bookparser.ChapterData{
			Title:       title,
			ContentPath: slidePath,
			Index:       i,
		})
	}
	return chapters, nil
}

func extractSlideNumber(name string) int {
	base := filepath.Base(name)
	re := regexp.MustCompile(`[0-9]+`)
	m := re.FindString(base)
	if m != "" {
		n, _ := strconv.Atoi(m)
		return n
	}
	return 0
}

func (p *Parser) extractPPTXSlideTitle(r *zip.ReadCloser, slidePath string) string {
	for _, f := range r.File {
		if f.Name == slidePath {
			rc, err := f.Open()
			if err != nil {
				return ""
			}
			defer rc.Close()

			decoder := xml.NewDecoder(rc)
			for {
				tok, err := decoder.Token()
				if err != nil {
					break
				}
				if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "t" {
					var text string
					if decoder.DecodeElement(&text, &se) == nil {
						clean := strings.TrimSpace(text)
						if clean != "" {
							if len(clean) > 40 {
								clean = clean[:40] + "..."
							}
							return clean
						}
					}
				}
			}
		}
	}
	return ""
}

func (p *Parser) getPPTXSlideContent(filePath, contentPath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open pptx: %w", err)
	}
	defer r.Close()

	// Parse relationships for image embeds
	relMap := make(map[string]string)
	relsPath := fmt.Sprintf("ppt/slides/_rels/%s.rels", filepath.Base(contentPath))
	for _, f := range r.File {
		if f.Name == relsPath {
			if rc, err := f.Open(); err == nil {
				var rels struct {
					Items []struct {
						ID     string `xml:"Id,attr"`
						Target string `xml:"Target,attr"`
					} `xml:"Relationship"`
				}
				if xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&rels) == nil {
					for _, item := range rels.Items {
						target := item.Target
						if strings.HasPrefix(target, "/") {
							if rel, relErr := filepath.Rel(filepath.Dir(contentPath), strings.TrimPrefix(target, "/")); relErr == nil {
								target = rel
							}
						}
						relMap[item.ID] = filepath.ToSlash(filepath.Clean(target))
					}
				}
				rc.Close()
			}
			break
		}
	}

	var slideFile *zip.File
	for _, f := range r.File {
		if f.Name == contentPath {
			slideFile = f
			break
		}
	}
	if slideFile == nil {
		return "", fmt.Errorf("slide %q not found", contentPath)
	}

	rc, err := slideFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var out strings.Builder
	out.WriteString(`<article class="novelhub-presentation-slide">`)

	var currentParagraph strings.Builder
	var currentAlign string
	var inRun bool
	var isBold, isItalic, isUnderline, isStrike bool
	var runText strings.Builder

	flushRun := func() {
		if inRun && runText.Len() > 0 {
			escaped := html.EscapeString(runText.String())
			if isBold {
				escaped = "<b>" + escaped + "</b>"
			}
			if isItalic {
				escaped = "<i>" + escaped + "</i>"
			}
			if isUnderline {
				escaped = "<u>" + escaped + "</u>"
			}
			if isStrike {
				escaped = "<s>" + escaped + "</s>"
			}
			currentParagraph.WriteString(escaped)
			runText.Reset()
		}
	}

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				currentParagraph.Reset()
				currentAlign = ""
			case "pPr":
				for _, attr := range t.Attr {
					if attr.Name.Local == "algn" {
						switch attr.Value {
						case "ctr":
							currentAlign = "center"
						case "r":
							currentAlign = "right"
						case "just":
							currentAlign = "justify"
						case "l":
							currentAlign = "left"
						}
					}
				}
			case "r":
				inRun = true
				isBold, isItalic, isUnderline, isStrike = false, false, false, false
				runText.Reset()
			case "rPr":
				if inRun {
					for _, attr := range t.Attr {
						switch attr.Name.Local {
						case "b":
							if attr.Value == "1" || attr.Value == "true" {
								isBold = true
							}
						case "i":
							if attr.Value == "1" || attr.Value == "true" {
								isItalic = true
							}
						case "u":
							if attr.Value != "none" && attr.Value != "0" {
								isUnderline = true
							}
						case "strike":
							if attr.Value != "noStrike" && attr.Value != "0" {
								isStrike = true
							}
						}
					}
				}
			case "t":
				var text string
				if decoder.DecodeElement(&text, &t) == nil && inRun {
					runText.WriteString(text)
				}
			case "blip", "videoFile", "audioFile", "quickTimeFile", "media":
				var embedID string
				for _, attr := range t.Attr {
					if attr.Name.Local == "embed" || attr.Name.Local == "link" || attr.Name.Local == "id" {
						embedID = attr.Value
					}
				}
				if embedID != "" {
					if mediaPath, ok := relMap[embedID]; ok {
						renderMediaElement(&out, mediaPath)
					}
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "r":
				flushRun()
				inRun = false
			case "p":
				flushRun()
				pText := strings.TrimSpace(currentParagraph.String())
				if pText != "" {
					alignAttr := ""
					if currentAlign != "" {
						alignAttr = fmt.Sprintf(` align="%s"`, currentAlign)
					}
					out.WriteString(fmt.Sprintf("<p%s style=\"margin: 0.5em 0;\">%s</p>\n", alignAttr, pText))
				}
			}
		}
	}

	out.WriteString("</article>")
	return out.String(), nil
}

// ----------------------------------------------------------------------------
// ODP (OpenDocument Presentation) Support
// ----------------------------------------------------------------------------

func (p *Parser) parseODPMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return p.defaultMetadata(filePath), nil
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "meta.xml" {
			rc, err := f.Open()
			if err == nil {
				var metaDoc struct {
					Title       string `xml:"meta>title"`
					Creator     string `xml:"meta>initial-creator"`
					Description string `xml:"meta>description"`
				}
				if xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&metaDoc) == nil {
					if t := strings.TrimSpace(metaDoc.Title); t != "" {
						meta.Title = t
					}
					meta.Author = strings.TrimSpace(metaDoc.Creator)
					meta.Description = strings.TrimSpace(metaDoc.Description)
				}
				rc.Close()
			}
			break
		}
	}

	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			meta.CoverType = "image/jpeg"
		}
	}

	merged := bookparser.MergeMetadataSidecar(filePath, meta)
	if len(merged.CoverData) == 0 {
		merged.CoverData = defaultcover.GenerateSVG(merged.Title, merged.Author)
		merged.IsDefaultCover = true
		merged.CoverType = "image/svg+xml"
	}
	return merged, nil
}

func (p *Parser) parseODPSpine(filePath string) ([]bookparser.ChapterData, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open odp: %w", err)
	}
	defer r.Close()

	var contentFile *zip.File
	for _, f := range r.File {
		if f.Name == "content.xml" {
			contentFile = f
			break
		}
	}
	if contentFile == nil {
		return p.defaultSpine(filePath), nil
	}

	rc, err := contentFile.Open()
	if err != nil {
		return p.defaultSpine(filePath), nil
	}
	defer rc.Close()

	var chapters []bookparser.ChapterData
	decoder := xml.NewDecoder(rc)
	pageIdx := 0

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "page" {
			pageName := ""
			for _, attr := range se.Attr {
				if attr.Name.Local == "name" {
					pageName = attr.Value
				}
			}
			if pageName == "" {
				pageName = fmt.Sprintf("Slide %d", pageIdx+1)
			}
			chapters = append(chapters, bookparser.ChapterData{
				Title:       pageName,
				ContentPath: fmt.Sprintf("page:%d", pageIdx),
				Index:       pageIdx,
			})
			pageIdx++
		}
	}

	if len(chapters) == 0 {
		return p.defaultSpine(filePath), nil
	}
	return chapters, nil
}



type odpStyle struct {
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Align     string
}

func parseODPStyles(data []byte) map[string]odpStyle {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	styles := make(map[string]odpStyle)

	var currentStyleName string
	var currentStyle odpStyle
	inStyle := false

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "style" {
				inStyle = true
				currentStyle = odpStyle{}
				for _, attr := range t.Attr {
					if attr.Name.Local == "name" {
						currentStyleName = attr.Value
					}
				}
			} else if inStyle && t.Name.Local == "text-properties" {
				for _, attr := range t.Attr {
					val := strings.ToLower(attr.Value)
					switch attr.Name.Local {
					case "font-weight", "font-weight-asian", "font-weight-complex":
						if val == "bold" || val == "700" || val == "800" || val == "900" {
							currentStyle.Bold = true
						}
					case "font-style", "font-style-asian", "font-style-complex":
						if val == "italic" || val == "oblique" {
							currentStyle.Italic = true
						}
					case "text-underline-style", "text-underline-type":
						if val != "none" && val != "" {
							currentStyle.Underline = true
						}
					case "text-line-through-style", "text-line-through-type":
						if val != "none" && val != "" {
							currentStyle.Strike = true
						}
					}
				}
			} else if inStyle && t.Name.Local == "paragraph-properties" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "text-align" {
						val := strings.ToLower(attr.Value)
						switch val {
						case "center":
							currentStyle.Align = "center"
						case "right", "end":
							currentStyle.Align = "right"
						case "justify":
							currentStyle.Align = "justify"
						case "left", "start":
							currentStyle.Align = "left"
						}
					}
				}
			}
		case xml.EndElement:
			if t.Name.Local == "style" && inStyle {
				if currentStyleName != "" {
					styles[currentStyleName] = currentStyle
				}
				inStyle = false
				currentStyleName = ""
			}
		}
	}
	return styles
}

func (p *Parser) getODPSlideContent(filePath, contentPath string) (string, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	targetIdx, err := strconv.Atoi(strings.TrimPrefix(contentPath, "page:"))
	if err != nil {
		targetIdx = 0
	}

	var contentFile *zip.File
	for _, f := range zr.File {
		if f.Name == "content.xml" {
			contentFile = f
			break
		}
	}
	if contentFile == nil {
		return "", fmt.Errorf("content.xml not found in odp")
	}

	rc, err := contentFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	contentBytes, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	styles := parseODPStyles(contentBytes)
	decoder := xml.NewDecoder(bytes.NewReader(contentBytes))
	currentPage := -1
	inTargetPage := false
	var out strings.Builder
	var block strings.Builder
	blockTag := ""
	blockAlign := ""
	inBlock := false
	var spanStyleStack []odpStyle

	out.WriteString(`<article class="novelhub-presentation-slide">`)
	flush := func() {
		value := strings.TrimSpace(block.String())
		if value != "" {
			out.WriteByte('<')
			out.WriteString(blockTag)
			if blockAlign != "" {
				out.WriteString(fmt.Sprintf(` align="%s"`, blockAlign))
			}
			out.WriteByte('>')
			out.WriteString(value)
			out.WriteString("</")
			out.WriteString(blockTag)
			out.WriteString(">\n")
		}
		block.Reset()
		blockTag = ""
		blockAlign = ""
		inBlock = false
	}

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "page" {
				currentPage++
				inTargetPage = (currentPage == targetIdx)
			}
			if inTargetPage {
				switch t.Name.Local {
				case "p", "h":
					if inBlock {
						flush()
					}
					blockTag = "p"
					for _, attr := range t.Attr {
						if attr.Name.Local == "style-name" {
							if s, ok := styles[attr.Value]; ok && s.Align != "" {
								blockAlign = s.Align
							}
						}
					}
					inBlock = true
				case "span":
					if inBlock {
						var curStyle odpStyle
						for _, attr := range t.Attr {
							if attr.Name.Local == "style-name" {
								if s, ok := styles[attr.Value]; ok {
									curStyle = s
								}
							}
						}
						spanStyleStack = append(spanStyleStack, curStyle)
						if curStyle.Bold {
							block.WriteString("<b>")
						}
						if curStyle.Italic {
							block.WriteString("<i>")
						}
						if curStyle.Underline {
							block.WriteString("<u>")
						}
						if curStyle.Strike {
							block.WriteString("<s>")
						}
					}
				case "image", "plugin", "object":
					if inBlock {
						flush()
					}
					for _, attr := range t.Attr {
						if attr.Name.Local == "href" {
							mediaPath := strings.TrimPrefix(attr.Value, "./")
							renderMediaElement(&out, mediaPath)
						}
					}
				case "line-break":
					if inBlock {
						block.WriteString("<br>")
					}
				}
			}
		case xml.CharData:
			if inTargetPage && inBlock {
				block.WriteString(html.EscapeString(string(t)))
			}
		case xml.EndElement:
			if inTargetPage {
				switch t.Name.Local {
				case "span":
					if inBlock && len(spanStyleStack) > 0 {
						top := spanStyleStack[len(spanStyleStack)-1]
						spanStyleStack = spanStyleStack[:len(spanStyleStack)-1]
						if top.Strike {
							block.WriteString("</s>")
						}
						if top.Underline {
							block.WriteString("</u>")
						}
						if top.Italic {
							block.WriteString("</i>")
						}
						if top.Bold {
							block.WriteString("</b>")
						}
					}
				case "p", "h":
					if inBlock {
						flush()
					}
				case "page":
					if inBlock {
						flush()
					}
					inTargetPage = false
				}
			}
		}
	}

	if inBlock {
		flush()
	}

	out.WriteString("</article>")
	return out.String(), nil
}

// ----------------------------------------------------------------------------
// Binary PPT (PowerPoint 97-2003) Support
// ----------------------------------------------------------------------------

func (p *Parser) parsePPTMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{
		Title:       bookparser.TitleFromPath(filePath),
		Description: fmt.Sprintf("PowerPoint Presentation: %s", filepath.Base(filePath)),
	}
	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			meta.CoverType = "image/jpeg"
		}
	}
	merged := bookparser.MergeMetadataSidecar(filePath, meta)
	if len(merged.CoverData) == 0 {
		merged.CoverData = defaultcover.GenerateSVG(merged.Title, merged.Author)
		merged.IsDefaultCover = true
		merged.CoverType = "image/svg+xml"
	}
	return merged, nil
}

func (p *Parser) parsePPTSpine(filePath string) ([]bookparser.ChapterData, error) {
	streams, err := readCompoundStreams(filePath)
	if err != nil {
		return nil, err
	}
	pptStream := streams["PowerPoint Document"]
	if len(pptStream) == 0 {
		return p.defaultSpine(filePath), nil
	}

	slides := extractPPTSlideContainers(pptStream)
	if len(slides) <= 1 {
		return []bookparser.ChapterData{{
			Title:       bookparser.TitleFromPath(filePath),
			ContentPath: "presentation",
			Index:       0,
		}}, nil
	}

	chapters := make([]bookparser.ChapterData, 0, len(slides))
	for i := range slides {
		chapters = append(chapters, bookparser.ChapterData{
			Title:       fmt.Sprintf("Slide %d", i+1),
			ContentPath: fmt.Sprintf("ppt-slide:%d", i),
			Index:       i,
		})
	}
	return chapters, nil
}

func (p *Parser) getPPTContent(filePath, contentPath string) (string, error) {
	streams, err := readCompoundStreams(filePath)
	if err != nil {
		return "", err
	}
	pptStream := streams["PowerPoint Document"]
	if len(pptStream) == 0 {
		return "<article><p>No presentation stream found in PPT.</p></article>", nil
	}

	var slideData []byte
	if strings.HasPrefix(contentPath, "ppt-slide:") {
		idxStr := strings.TrimPrefix(contentPath, "ppt-slide:")
		idx, convErr := strconv.Atoi(idxStr)
		if convErr != nil || idx < 0 {
			return "", fmt.Errorf("invalid ppt slide path %q", contentPath)
		}
		slides := extractPPTSlideContainers(pptStream)
		if idx >= len(slides) {
			return "", fmt.Errorf("ppt slide %d not found", idx)
		}
		slideData = slides[idx]
	} else {
		slideData = pptStream
	}

	texts := extractPPTTextRecords(slideData)
	if len(texts) == 0 {
		return "<article><p>No readable slide text found in PPT file.</p></article>", nil
	}

	var out strings.Builder
	out.WriteString(`<article class="novelhub-presentation-slide">`)
	for _, txt := range texts {
		txt = cleanPPTString(txt)
		if txt == "" {
			continue
		}
		lines := strings.Split(txt, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && isMeaningfulText(line) {
				fmt.Fprintf(&out, "<p>%s</p>\n", html.EscapeString(line))
			}
		}
	}
	out.WriteString("</article>")
	return out.String(), nil
}

// extractPPTSlideContainers returns the byte ranges of the top-level slide
// containers (RT_SLIDE = 1006). Text records (RT_TextCharsAtom etc.) live
// inside these containers, so extracting per-container yields per-slide text.
// Main masters use RT_MAIN_MASTER = 1016 and are excluded.
func extractPPTSlideContainers(data []byte) [][]byte {
	var slides [][]byte
	pos := 0
	for pos+8 <= len(data) {
		recVer := data[pos] & 0x0F
		recType := binary.LittleEndian.Uint16(data[pos+2 : pos+4])
		recLen := binary.LittleEndian.Uint32(data[pos+4 : pos+8])

		bodyStart := pos + 8
		if int64(bodyStart)+int64(recLen) > int64(len(data)) {
			break
		}
		if recVer == 0x0F && recType == 1006 {
			slides = append(slides, data[bodyStart:bodyStart+int(recLen)])
		}
		pos += 8 + int(recLen)
	}
	return slides
}

func cleanPPTString(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\r' || r == '\n' {
			b.WriteByte('\n')
		} else if r >= 0x20 || r == '\t' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func isMeaningfulText(s string) bool {
	if len(s) == 0 {
		return false
	}
	hasLetterOrDigit := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r >= 0x80 {
			hasLetterOrDigit = true
			break
		}
	}
	return hasLetterOrDigit
}

func readCompoundStreams(filePath string) (map[string][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open compound file: %w", err)
	}
	defer file.Close()

	reader, err := mscfb.New(file)
	if err != nil {
		return nil, fmt.Errorf("open compound reader: %w", err)
	}
	streams := make(map[string][]byte)
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || entry.Size <= 0 {
			continue
		}
		data, err := bookparser.ReadAllLimit(entry, constants.MaxArchiveAssetSize)
		if err != nil {
			continue
		}
		streams[entry.Name] = data
	}
	return streams, nil
}

func extractPPTTextRecords(data []byte) []string {
	var results []string
	pos := 0
	for pos+8 <= len(data) {
		recVer := data[pos] & 0x0F
		recType := binary.LittleEndian.Uint16(data[pos+2 : pos+4])
		recLen := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		pos += 8

		if pos+int(recLen) > len(data) {
			break
		}

		// RT_TextCharsAtom = 4000 (UTF-16)
		if recType == 4000 && recLen > 0 {
			u16 := make([]uint16, recLen/2)
			for i := range u16 {
				u16[i] = binary.LittleEndian.Uint16(data[pos+i*2 : pos+i*2+2])
			}
			txt := strings.TrimSpace(string(utf16.Decode(u16)))
			if txt != "" {
				results = append(results, txt)
			}
		} else if (recType == 4008 || recType == 4010) && recLen > 0 { // RT_TextBytesAtom / RT_CString (ASCII)
			txt := strings.TrimSpace(string(data[pos : pos+int(recLen)]))
			if txt != "" {
				results = append(results, txt)
			}
		}

		if recVer != 0x0F {
			pos += int(recLen)
		}
	}
	return results
}

func renderMediaElement(out *strings.Builder, mediaPath string) {
	cleanPath := html.EscapeString(mediaPath)
	ext := strings.ToLower(filepath.Ext(mediaPath))

	switch ext {
	case ".mp3", ".m4a", ".wav", ".ogg", ".aac", ".wma", ".flac", ".opus", ".mid", ".midi":
		fmt.Fprintf(out, `<div class="slide-audio" style="margin: 1em 0; text-align: center;"><audio controls preload="metadata" src="%s" style="max-width: 100%%; width: 360px;"></audio></div>`, cleanPath)
	case ".mp4", ".webm", ".m4v", ".ogv", ".avi", ".mov", ".wmv", ".mkv", ".flv":
		fmt.Fprintf(out, `<div class="slide-video" style="margin: 1em 0; text-align: center;"><video controls preload="metadata" playsinline src="%s" style="max-width: 100%%; max-height: 480px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.15);"></video></div>`, cleanPath)
	default:
		fmt.Fprintf(out, `<div class="slide-image" style="margin: 1em 0; text-align: center;"><img src="%s" style="max-width: 100%%; height: auto; border-radius: 8px;" /></div>`, cleanPath)
	}
}
