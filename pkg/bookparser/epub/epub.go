package epub

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

type Container struct {
	XMLName   xml.Name   `xml:"container"`
	Rootfiles []Rootfile `xml:"rootfiles>rootfile"`
}

type Rootfile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

type Package struct {
	XMLName  xml.Name `xml:"package"`
	Metadata Metadata `xml:"metadata"`
	Manifest Manifest `xml:"manifest"`
	Spine    Spine    `xml:"spine"`
	Guide    Guide    `xml:"guide"`
}

type Metadata struct {
	Titles       []TextValue  `xml:"title"`
	Creators     []TextValue  `xml:"creator"`
	Contributors []TextValue  `xml:"contributor"`
	Descriptions []TextValue  `xml:"description"`
	Publishers   []TextValue  `xml:"publisher"`
	Languages    []TextValue  `xml:"language"`
	Dates        []TextValue  `xml:"date"`
	Subjects     []TextValue  `xml:"subject"`
	Identifiers  []Identifier `xml:"identifier"`
	Meta         []Meta       `xml:"meta"`
}

type TextValue struct {
	ID     string `xml:"id,attr" json:"id,omitempty"`
	FileAs string `xml:"file-as,attr" json:"fileAs,omitempty"`
	Role   string `xml:"role,attr" json:"role,omitempty"`
	Value  string `xml:",chardata" json:"value,omitempty"`
}

type Identifier struct {
	ID     string `xml:"id,attr" json:"id,omitempty"`
	Scheme string `xml:"scheme,attr" json:"scheme,omitempty"`
	Value  string `xml:",chardata" json:"value,omitempty"`
}

type Meta struct {
	ID       string `xml:"id,attr" json:"id,omitempty"`
	Name     string `xml:"name,attr" json:"name,omitempty"`
	Content  string `xml:"content,attr" json:"content,omitempty"`
	Property string `xml:"property,attr" json:"property,omitempty"`
	Refines  string `xml:"refines,attr" json:"refines,omitempty"`
	Scheme   string `xml:"scheme,attr" json:"scheme,omitempty"`
	Value    string `xml:",chardata" json:"value,omitempty"`
}

type NormalizedMetadata struct {
	Title        string       `json:"title,omitempty"`
	Creator      string       `json:"creator,omitempty"`
	Creators     []string     `json:"creators,omitempty"`
	Contributors []string     `json:"contributors,omitempty"`
	Description  string       `json:"description,omitempty"`
	Publisher    string       `json:"publisher,omitempty"`
	Publishers   []string     `json:"publishers,omitempty"`
	Language     string       `json:"language,omitempty"`
	Languages    []string     `json:"languages,omitempty"`
	Date         string       `json:"date,omitempty"`
	Dates        []string     `json:"dates,omitempty"`
	Subject      []string     `json:"subject,omitempty"`
	Identifier   []Identifier `json:"identifier,omitempty"`
	Meta         []Meta       `json:"meta,omitempty"`
	Series       string       `json:"series,omitempty"`
	SeriesIndex  string       `json:"seriesIndex,omitempty"`
}

type Manifest struct {
	Items []Item `xml:"item"`
}

type Item struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type Spine struct {
	TOC      string    `xml:"toc,attr"`
	Itemrefs []Itemref `xml:"itemref"`
}

type Itemref struct {
	IDRef string `xml:"idref,attr"`
}

type Guide struct {
	References []GuideReference `xml:"reference"`
}

type GuideReference struct {
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
	Href  string `xml:"href,attr"`
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return nil, err
	}

	opfFile, err := getZipFile(r, opfPath)
	if err != nil {
		return nil, err
	}

	rc, err := opfFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var pkg Package
	if err := xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&pkg); err != nil {
		return nil, err
	}

	normalized := normalizeMetadata(pkg.Metadata)
	title := normalized.Title
	if strings.TrimSpace(title) == "" {
		title = bookparser.TitleFromPath(filePath)
	}
	normalized.Title = title
	meta := &bookparser.BookMetadata{
		Title:       title,
		Author:      normalized.Creator,
		Description: normalized.Description,
		Publisher:   normalized.Publisher,
		Language:    normalized.Language,
		Date:        normalized.Date,
		Subjects:    normalized.Subject,
		Series:      normalized.Series,
		SeriesIndex: normalized.SeriesIndex,
	}

	meta.MetadataJSON, _ = jsonx.MarshalString(normalized)

	var coverHref string
	var coverID string

	for _, m := range pkg.Metadata.Meta {
		if strings.EqualFold(m.Name, "cover") || strings.EqualFold(m.Property, "cover") {
			coverID = cleanMetadataText(m.Content)
			if coverID == "" {
				coverID = cleanMetadataText(m.Value)
			}
			break
		}
	}

	for _, item := range pkg.Manifest.Items {
		if coverID != "" && item.ID == coverID && strings.HasPrefix(item.MediaType, "image/") {
			coverHref = item.Href
			meta.CoverType = item.MediaType
			break
		}
	}
	if coverHref == "" {
		for _, item := range pkg.Manifest.Items {
			if strings.Contains(strings.ToLower(item.Properties), "cover-image") && strings.HasPrefix(item.MediaType, "image/") {
				coverHref = item.Href
				meta.CoverType = item.MediaType
				break
			}
		}
	}
	if coverHref == "" {
		for _, item := range pkg.Manifest.Items {
			itemID := strings.ToLower(item.ID)
			itemHref := strings.ToLower(item.Href)
			if strings.HasPrefix(item.MediaType, "image/") && (strings.Contains(itemID, "cover") || strings.Contains(itemHref, "cover")) {
				coverHref = item.Href
				meta.CoverType = item.MediaType
				break
			}
		}
	}
	if coverHref == "" {
		for _, ref := range pkg.Guide.References {
			if strings.EqualFold(ref.Type, "cover") && strings.TrimSpace(ref.Href) != "" {
				if item, ok := findImageManifestItemByHref(pkg.Manifest.Items, ref.Href); ok {
					coverHref = item.Href
					meta.CoverType = item.MediaType
					break
				}
			}
		}
	}
	if coverHref == "" {
		for _, item := range pkg.Manifest.Items {
			if strings.HasPrefix(item.MediaType, "image/") {
				coverHref = item.Href
				meta.CoverType = item.MediaType
				break
			}
		}
	}

	if coverHref != "" {
		baseDir := filepath.Dir(opfPath)
		if baseDir == "." {
			baseDir = ""
		}
		coverPath := resolveEPUBHref(baseDir, coverHref)

		coverFile, err := getZipFile(r, coverPath)
		if err == nil {
			crc, err := coverFile.Open()
			if err == nil {
				data, err := bookparser.ReadAllLimit(crc, constants.MaxCoverBytes)
				if err == nil {
					meta.CoverData = data
				}
				_ = crc.Close()
			}
		}
	}

	return meta, nil
}

func normalizeMetadata(metadata Metadata) NormalizedMetadata {
	creators := textValues(metadata.Creators)
	publishers := textValues(metadata.Publishers)
	languages := textValues(metadata.Languages)
	dates := textValues(metadata.Dates)
	series, seriesIndex := extractSeries(metadata.Meta)

	return NormalizedMetadata{
		Title:        firstTextValue(metadata.Titles),
		Creator:      strings.Join(creators, ", "),
		Creators:     creators,
		Contributors: textValues(metadata.Contributors),
		Description:  strings.Join(textValues(metadata.Descriptions), "\n\n"),
		Publisher:    strings.Join(publishers, ", "),
		Publishers:   publishers,
		Language:     strings.Join(languages, ", "),
		Languages:    languages,
		Date:         firstTextValue(metadata.Dates),
		Dates:        dates,
		Subject:      textValues(metadata.Subjects),
		Identifier:   normalizeIdentifiers(metadata.Identifiers),
		Meta:         normalizeMeta(metadata.Meta),
		Series:       series,
		SeriesIndex:  seriesIndex,
	}
}

func textValues(values []TextValue) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		value := cleanMetadataText(item.Value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func firstTextValue(values []TextValue) string {
	list := textValues(values)
	if len(list) == 0 {
		return ""
	}
	return list[0]
}

func cleanMetadataText(value string) string {
	return bookparser.CleanChapterTitle(value)
}

func normalizeIdentifiers(items []Identifier) []Identifier {
	result := make([]Identifier, 0, len(items))
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Scheme = strings.TrimSpace(item.Scheme)
		item.Value = cleanMetadataText(item.Value)
		if item.ID == "" && item.Scheme == "" && item.Value == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func normalizeMeta(items []Meta) []Meta {
	result := make([]Meta, 0, len(items))
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Name = strings.TrimSpace(item.Name)
		item.Content = cleanMetadataText(item.Content)
		item.Property = strings.TrimSpace(item.Property)
		item.Refines = strings.TrimSpace(item.Refines)
		item.Scheme = strings.TrimSpace(item.Scheme)
		item.Value = cleanMetadataText(item.Value)
		if item.ID == "" && item.Name == "" && item.Content == "" && item.Property == "" && item.Value == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func extractSeries(items []Meta) (string, string) {
	var series string
	var seriesIndex string
	for _, item := range items {
		name := strings.ToLower(strings.TrimSpace(item.Name))
		property := strings.ToLower(strings.TrimSpace(item.Property))
		content := cleanMetadataText(item.Content)
		value := cleanMetadataText(item.Value)
		if content == "" {
			content = value
		}
		switch {
		case name == "calibre:series" && content != "":
			series = content
		case name == "calibre:series_index" && content != "":
			seriesIndex = content
		case property == "belongs-to-collection" && content != "":
			series = content
		case property == "group-position" && content != "":
			seriesIndex = content
		}
	}
	return series, seriesIndex
}

func findImageManifestItemByHref(items []Item, href string) (Item, bool) {
	target := strings.TrimLeft(resolveEPUBHref("", href), "/")
	for _, item := range items {
		if !strings.HasPrefix(item.MediaType, "image/") {
			continue
		}
		itemHref := strings.TrimLeft(resolveEPUBHref("", item.Href), "/")
		if itemHref == target || strings.HasSuffix(target, "/"+itemHref) || strings.HasSuffix(itemHref, "/"+target) {
			return item, true
		}
	}
	return Item{}, false
}

type NCX struct {
	XMLName xml.Name `xml:"ncx"`
	NavMap  NavMap   `xml:"navMap"`
}

type NavMap struct {
	NavPoints []NavPoint `xml:"navPoint"`
}

type NavPoint struct {
	NavLabel  NavLabel   `xml:"navLabel"`
	Content   NCXContent `xml:"content"`
	NavPoints []NavPoint `xml:"navPoint"`
}

type NavLabel struct {
	Text string `xml:"text"`
}

type NCXContent struct {
	Src string `xml:"src,attr"`
}

func parseNCXTOC(r *zip.ReadCloser, ncxPath string) map[string]string {
	tocMap := make(map[string]string)
	ncxFile, err := getZipFile(r, ncxPath)
	if err != nil {
		return tocMap
	}
	rc, err := ncxFile.Open()
	if err != nil {
		return tocMap
	}
	defer rc.Close()
	var ncx NCX
	if err := xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&ncx); err != nil {
		return tocMap
	}
	ncxBaseDir := filepath.Dir(ncxPath)
	var walk func(points []NavPoint)
	walk = func(points []NavPoint) {
		for _, p := range points {
			text := bookparser.CleanChapterTitle(p.NavLabel.Text)
			src := strings.TrimSpace(p.Content.Src)
			if text != "" && src != "" {
				cleanSrc := strings.Split(src, "#")[0]
				resolved := resolveEPUBHref(ncxBaseDir, cleanSrc)
				if _, exists := tocMap[resolved]; !exists {
					tocMap[resolved] = text
				}
			}
			if len(p.NavPoints) > 0 {
				walk(p.NavPoints)
			}
		}
	}
	walk(ncx.NavMap.NavPoints)
	return tocMap
}

func parseEPUB3NavTOC(r *zip.ReadCloser, navPath string) map[string]string {
	tocMap := make(map[string]string)
	navFile, err := getZipFile(r, navPath)
	if err != nil {
		return tocMap
	}
	rc, err := navFile.Open()
	if err != nil {
		return tocMap
	}
	defer rc.Close()
	data, err := bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
	if err != nil {
		return tocMap
	}
	navBaseDir := filepath.Dir(navPath)
	aRegex := regexp.MustCompile(`(?i)<a[^>]+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	matches := aRegex.FindAllStringSubmatch(string(data), -1)
	tagRegexp := regexp.MustCompile(`<[^>]*>`)
	for _, m := range matches {
		if len(m) >= 3 {
			href := strings.TrimSpace(m[1])
			text := bookparser.CleanChapterTitle(tagRegexp.ReplaceAllString(m[2], ""))
			if text != "" && href != "" {
				cleanHref := strings.Split(href, "#")[0]
				resolved := resolveEPUBHref(navBaseDir, cleanHref)
				if _, exists := tocMap[resolved]; !exists {
					tocMap[resolved] = text
				}
			}
		}
	}
	return tocMap
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return nil, err
	}
	opfFile, err := getZipFile(r, opfPath)
	if err != nil {
		return nil, err
	}
	rc, err := opfFile.Open()
	if err != nil {
		return nil, err
	}
	var pkg Package
	_ = xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&pkg)
	_ = rc.Close()

	baseDir := filepath.Dir(opfPath)
	if baseDir == "." {
		baseDir = ""
	}

	itemsMap := make(map[string]Item)
	var ncxPath string
	var navPath string

	for _, item := range pkg.Manifest.Items {
		itemsMap[item.ID] = item
		if item.MediaType == "application/x-dtbncx+xml" || item.ID == pkg.Spine.TOC {
			ncxPath = resolveEPUBHref(baseDir, item.Href)
		}
		if strings.Contains(item.Properties, "nav") {
			navPath = resolveEPUBHref(baseDir, item.Href)
		}
	}

	tocMap := make(map[string]string)
	if ncxPath != "" {
		tocMap = parseNCXTOC(r, ncxPath)
	}
	if len(tocMap) == 0 && navPath != "" {
		tocMap = parseEPUB3NavTOC(r, navPath)
	}

	coverHrefs := make(map[string]struct{})
	for _, ref := range pkg.Guide.References {
		refType := strings.ToLower(ref.Type)
		if refType == "cover" || refType == "title-page" || refType == "titlepage" {
			resolved := resolveEPUBHref(baseDir, ref.Href)
			coverHrefs[resolved] = struct{}{}
		}
	}

	chapters := []bookparser.ChapterData{}
	index := 0
	for _, ref := range pkg.Spine.Itemrefs {
		item, ok := itemsMap[ref.IDRef]
		if !ok || !strings.Contains(item.MediaType, "html") {
			continue
		}

		chPath := resolveEPUBHref(baseDir, item.Href)
		var title string

		if t, ok := tocMap[chPath]; ok && t != "" {
			title = t
		} else if _, isCover := coverHrefs[chPath]; isCover || strings.Contains(strings.ToLower(item.ID), "titlepage") || strings.Contains(strings.ToLower(item.Href), "titlepage") {
			title = "Cover"
		} else {
			title = extractTitleFromZip(r, chPath)
		}

		if title == "" {
			base := strings.TrimSuffix(filepath.Base(item.Href), filepath.Ext(item.Href))
			base = strings.ReplaceAll(base, "_", " ")
			base = strings.ReplaceAll(base, "-", " ")
			title = base
		}

		title = bookparser.CleanChapterTitle(title)

		chapters = append(chapters, bookparser.ChapterData{
			Title:       title,
			ContentPath: chPath,
			Index:       index,
		})
		index++
	}

	return chapters, nil
}

var htmlTagRegexp = regexp.MustCompile(`<[^>]*>`)

func extractTitleFromZip(r *zip.ReadCloser, path string) string {
	f, err := getZipFile(r, path)
	if err != nil {
		return ""
	}
	rc, err := f.Open()
	if err != nil {
		return ""
	}
	defer rc.Close()

	buf := make([]byte, 8192)
	n, _ := rc.Read(buf)
	html := string(buf[:n])

	for _, tag := range []string{"h1", "h2", "h3"} {
		lowerHtml := strings.ToLower(html)
		openTag := "<" + tag
		start := strings.Index(lowerHtml, openTag)
		if start != -1 {
			gt := strings.Index(lowerHtml[start:], ">")
			if gt != -1 {
				startTagEnd := start + gt + 1
				closeTag := "</" + tag + ">"
				end := strings.Index(lowerHtml[startTagEnd:], closeTag)
				if end != -1 {
					rawText := html[startTagEnd : startTagEnd+end]
					cleanText := strings.TrimSpace(htmlTagRegexp.ReplaceAllString(rawText, ""))
					if cleanText != "" {
						return cleanText
					}
				}
			}
		}
	}

	start := strings.Index(strings.ToLower(html), "<title>")
	if start != -1 {
		start += len("<title>")
		end := strings.Index(strings.ToLower(html[start:]), "</title>")
		if end != -1 {
			title := strings.TrimSpace(html[start : start+end])
			lower := strings.ToLower(title)
			if title != "" && lower != "unknown" && lower != "untitled" && lower != "table of contents" {
				return title
			}
		}
	}
	return ""
}

func (p *Parser) ParseBook(filePath string) (*bookparser.BookData, error) {
	meta, err := p.ParseMetadata(filePath)
	if err != nil {
		return nil, err
	}

	spine, err := p.ParseSpine(filePath)
	if err != nil {
		return nil, err
	}

	for i := range spine {
		content, _ := p.GetChapterContent(filePath, spine[i].ContentPath)
		spine[i].Content = content
	}

	return &bookparser.BookData{
		Metadata: *meta,
		Chapters: spine,
	}, nil
}

func (p *Parser) GetChapterContent(filePath, contentPath string) (string, error) {
	data, err := p.GetAsset(filePath, contentPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	f, err := getZipFile(r, assetPath)
	if err != nil {
		return nil, err
	}

	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return nil, err
	}
	opfFile, err := getZipFile(r, opfPath)
	if err != nil {
		return nil, err
	}
	rc, err := opfFile.Open()
	if err != nil {
		return nil, err
	}
	var pkg Package
	_ = xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&pkg)
	_ = rc.Close()

	baseDir := filepath.Dir(opfPath)
	if baseDir == "." {
		baseDir = ""
	}

	var images []string
	for _, item := range pkg.Manifest.Items {
		if strings.HasPrefix(item.MediaType, "image/") {
			imgPath := filepath.Join(baseDir, item.Href)
			imgPath = resolveEPUBHref(baseDir, item.Href)
			images = append(images, imgPath)
		}
	}

	return images, nil
}

func findOPFPath(r *zip.ReadCloser) (string, error) {
	containerFile, err := getZipFile(r, "META-INF/container.xml")
	if err != nil {
		return "", errors.New("not a valid epub: missing container.xml")
	}

	rc, err := containerFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	var container Container
	if err := xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&container); err != nil {
		return "", err
	}

	for _, rf := range container.Rootfiles {
		if rf.MediaType == "application/oebps-package+xml" {
			return rf.FullPath, nil
		}
	}

	return "", errors.New("no OPF file found")
}

func getZipFile(r *zip.ReadCloser, name string) (*zip.File, error) {
	for _, f := range r.File {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, errors.New("file not found in zip: " + name)
}

func resolveEPUBHref(baseDir string, href string) string {
	href, _, _ = strings.Cut(strings.TrimSpace(href), "#")
	href, _, _ = strings.Cut(href, "?")
	if decoded, err := url.PathUnescape(href); err == nil {
		href = decoded
	}
	if baseDir == "." {
		baseDir = ""
	}
	value := filepath.Join(baseDir, href)
	return strings.ReplaceAll(value, "\\", "/")
}
