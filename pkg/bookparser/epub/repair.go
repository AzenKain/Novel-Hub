package epub

import (
	"archive/zip"
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ValidationIssue represents a detected problem in an EPUB file conforming to EPUB-Forge standard.
type ValidationIssue struct {
	Severity string `json:"severity"` // "error", "warning", "info"
	Code     string `json:"code"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
	Fixable  bool   `json:"fixable"`
	FixID    string `json:"fix_id,omitempty"`
}

// ValidationReport summarizes the health check of an EPUB file.
type ValidationReport struct {
	Valid    bool              `json:"valid"`
	Errors   int               `json:"errors"`
	Warnings int               `json:"warnings"`
	Infos    int               `json:"infos"`
	Issues   []ValidationIssue `json:"issues"`
}

// RepairOptions defines which automated fixes to apply to an EPUB file.
type RepairOptions struct {
	NormalizeMimetype bool     `json:"normalize_mimetype"` // PACKAGE_MIMETYPE
	FixContainer      bool     `json:"fix_container"`      // Ensure META-INF/container.xml points to OPF
	FixXHTML          bool     `json:"fix_xhtml"`          // FIX_XHTML: self-closing tags, XML entities, xmlns
	ReconcileManifest bool     `json:"reconcile_manifest"` // REMOVE_MISSING_MANIFEST_ITEMS, FIX_MEDIA_TYPES, ADD_UNMANIFESTED_FILES
	ReconcileSpine    bool     `json:"reconcile_spine"`    // Remove itemrefs pointing to missing manifest items
	FixTOC            bool     `json:"fix_toc"`            // FIX_TOC_NCX: Rebuild/repair toc.ncx and nav.xhtml
	CleanBrokenLinks  bool     `json:"clean_broken_links"`  // CLEAN_BROKEN_CONTENT_LINKS: Remove dangling local hrefs and missing img sources
	UpgradeEPUB3      bool     `json:"upgrade_epub3"`      // UPGRADE_EPUB3: Upgrade package version to 3.0, dcterms:modified, nav.xhtml
	BuildCoverPage    bool     `json:"build_cover_page"`   // BUILD_COVER_PAGE: Generate cover.xhtml from cover image
	BuildTOCPage      bool     `json:"build_toc_page"`     // BUILD_TOC_PAGE: Generate visible toc.xhtml in spine
	BuildTitles       bool     `json:"build_titles"`       // BUILD_CHAPTER_TITLES: Inject missing <h2> titles
	FixMetadata       bool     `json:"fix_metadata"`       // Ensure dc:title, dc:language, dcterms:modified
	FixList           []string `json:"fix_list,omitempty"` // Explicit list of Fix IDs
}

// DefaultRepairOptions returns the recommended auto-repair options.
func DefaultRepairOptions() RepairOptions {
	return RepairOptions{
		NormalizeMimetype: true,
		FixContainer:      true,
		FixXHTML:          true,
		ReconcileManifest: true,
		ReconcileSpine:    true,
		FixTOC:            true,
		CleanBrokenLinks:  true,
		UpgradeEPUB3:      true,
		BuildCoverPage:    true,
		BuildTOCPage:      true,
		BuildTitles:       true,
		FixMetadata:       true,
	}
}

func (opts RepairOptions) toFixSelection() repairSelection {
	sel := repairSelection{}
	if len(opts.FixList) > 0 {
		for _, fix := range opts.FixList {
			fix = strings.TrimSpace(fix)
			if fix != "" {
				sel[fix] = true
			}
		}
		return sel
	}

	if opts.NormalizeMimetype {
		sel["PACKAGE_MIMETYPE"] = true
	}
	if opts.FixXHTML {
		sel["FIX_XHTML"] = true
	}
	if opts.ReconcileManifest {
		sel["REMOVE_MISSING_MANIFEST_ITEMS"] = true
		sel["FIX_MEDIA_TYPES"] = true
		sel["ADD_UNMANIFESTED_FILES"] = true
	}
	if opts.ReconcileSpine {
		sel["REMOVE_MISSING_MANIFEST_ITEMS"] = true
	}
	if opts.FixTOC {
		sel["FIX_TOC_NCX"] = true
	}
	if opts.CleanBrokenLinks {
		sel["CLEAN_BROKEN_CONTENT_LINKS"] = true
	}
	if opts.UpgradeEPUB3 {
		sel["UPGRADE_EPUB3"] = true
	}
	if opts.BuildCoverPage {
		sel["BUILD_COVER_PAGE"] = true
	}
	if opts.BuildTOCPage {
		sel["BUILD_TOC_PAGE"] = true
	}
	if opts.BuildTitles {
		sel["BUILD_CHAPTER_TITLES"] = true
	}

	return sel
}

// RepairResult contains the summary and logs of a repair operation.
type RepairResult struct {
	Success    bool             `json:"success"`
	FixedCount int              `json:"fixed_count"`
	Logs       []string         `json:"logs"`
	Report     ValidationReport `json:"report"`
}

type repairSelection map[string]bool

func (s repairSelection) empty() bool {
	return len(s) == 0
}

func (s repairSelection) has(fix string) bool {
	return s[fix]
}

// Internal Manifest, Spine & TOC structures mirroring EPUB-Forge
type ManifestItem struct {
	ID        string            `json:"id"`
	Href      string            `json:"href"`
	FullPath  string            `json:"full_path"`
	MediaType string            `json:"media_type"`
	Raw       string            `json:"raw,omitempty"`
	Attrs     map[string]string `json:"attrs,omitempty"`
}

type SpineRef struct {
	IDRef  string            `json:"idref"`
	Linear bool              `json:"linear"`
	Raw    string            `json:"raw,omitempty"`
	Attrs  map[string]string `json:"attrs,omitempty"`
}

type TocPoint struct {
	Title    string `json:"title"`
	Src      string `json:"src"`
	FullPath string `json:"full_path"`
}

type ChapterInfo struct {
	IDRef string `json:"idref"`
	Href  string `json:"href"`
	Path  string `json:"path"`
	Title string `json:"title"`
	Index int    `json:"index"`
}

type BookMetadata struct {
	Title       string `json:"title"`
	Creator     string `json:"creator"`
	Language    string `json:"language"`
	Publisher   string `json:"publisher"`
	Description string `json:"description"`
}

type BookContext struct {
	FilePath       string
	FileName       string
	Size           int64
	Reader         *zip.ReadCloser
	Entries        map[string]*zip.File
	OPFPath        string
	OPFDir         string
	OPFXML         string
	Manifest       []ManifestItem
	ManifestByID   map[string]ManifestItem
	ManifestByPath map[string]ManifestItem
	Spine          []SpineRef
	Chapters       []ChapterInfo
	Title          string
	Creator        string
	Metadata       BookMetadata
	NCX            *ManifestItem
	TOC            []TocPoint
}

func (ctx *BookContext) Close() {
	if ctx.Reader != nil {
		_ = ctx.Reader.Close()
	}
}

func (ctx *BookContext) readText(p string) (string, error) {
	f := ctx.Entries[p]
	if f == nil {
		return "", fmt.Errorf("file not found in zip: %s", p)
	}
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Regex definitions identical to EPUB-Forge
var (
	manifestRe    = regexp.MustCompile(`(?is)(<manifest\b[^>]*>)(.*?)(</manifest>)`)
	spineRe       = regexp.MustCompile(`(?is)(<spine\b[^>]*>)(.*?)(</spine>)`)
	attrRe        = regexp.MustCompile(`([\w:.-]+)\s*=\s*["']([^"']*)["']`)
	hrefAttrRe    = regexp.MustCompile(`(?is)<a\b[^>]*[\s]href\s*=\s*["']([^"']+)["']`)
	titleTextRe   = regexp.MustCompile(`(?is)<dc:title\b[^>]*>(.*?)</dc:title>`)
	creatorTextRe = regexp.MustCompile(`(?is)<dc:creator\b[^>]*>(.*?)</dc:creator>`)
	langTextRe    = regexp.MustCompile(`(?is)<dc:language\b[^>]*>(.*?)</dc:language>`)
)

func parseAttrs(tag string) map[string]string {
	attrs := make(map[string]string)
	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		if len(m) == 3 {
			attrs[m[1]] = m[2]
		}
	}
	return attrs
}

func posixDir(p string) string {
	d := path.Dir(p)
	if d == "." || d == "/" {
		return ""
	}
	return d + "/"
}

func normalizeZipPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return strings.TrimPrefix(path.Clean("/"+p), "/")
}

func relativeZipPath(from, to string) string {
	fromDir := path.Dir(from)
	if fromDir == "." || fromDir == "" {
		return to
	}
	rel, err := filepath.Rel(fromDir, to)
	if err != nil {
		return to
	}
	return filepath.ToSlash(rel)
}

func resolveZipHref(baseDir, href string) string {
	cleanHref := strings.Split(href, "#")[0]
	cleanHref = strings.Split(cleanHref, "?")[0]
	if cleanHref == "" {
		return ""
	}
	if unescaped, err := url.PathUnescape(cleanHref); err == nil {
		cleanHref = unescaped
	}
	if strings.HasPrefix(cleanHref, "/") {
		return normalizeZipPath(cleanHref[1:])
	}
	return normalizeZipPath(path.Join(baseDir, cleanHref))
}

func isExternalRef(ref string) bool {
	lower := strings.ToLower(strings.TrimSpace(ref))
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "ftp://") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "tel:")
}

func skipLocalReference(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "#") || isExternalRef(ref) {
		return true
	}
	lower := strings.ToLower(ref)
	return strings.HasPrefix(lower, "javascript:")
}

func hasPropertyToken(properties, token string) bool {
	for _, field := range strings.Fields(properties) {
		if field == token {
			return true
		}
	}
	return false
}

func stripTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

func cleanText(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.ReplaceAll(s, "\u200b", " ")
	return strings.Join(strings.Fields(s), " ")
}

func sanitizeManifestID(id string) string {
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			return r
		}
		return '_'
	}, id)
	if id == "" || (!((id[0] >= 'a' && id[0] <= 'z') || (id[0] >= 'A' && id[0] <= 'Z') || id[0] == '_')) {
		id = "item_" + id
	}
	return id
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func validateXMLWellFormed(source string) error {
	decoder := xml.NewDecoder(strings.NewReader(source))
	for {
		if _, err := decoder.Token(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func isChapterTitleMissing(htmlStr string) bool {
	reBody := regexp.MustCompile(`(?i)<body[^>]*>`)
	bodyLoc := reBody.FindStringIndex(htmlStr)
	var bodyContent string
	if bodyLoc == nil {
		bodyContent = htmlStr
	} else {
		bodyContent = htmlStr[bodyLoc[1]:]
	}

	reHeading := regexp.MustCompile(`(?i)<h[1-6]\b`)
	headingLoc := reHeading.FindStringIndex(bodyContent)
	if headingLoc == nil {
		return true // No heading at all
	}

	beforeHeading := bodyContent[:headingLoc[0]]
	stripped := stripTags(beforeHeading)
	unescaped := html.UnescapeString(stripped)
	unescaped = strings.ReplaceAll(unescaped, "\u00a0", " ")
	unescaped = strings.ReplaceAll(unescaped, "\u200b", " ")

	if strings.TrimSpace(unescaped) != "" {
		return true // Text exists before heading
	}
	return false
}

func contentTypeFor(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xhtml", ".html", ".htm":
		return "application/xhtml+xml"
	case ".css":
		return "text/css"
	case ".ncx":
		return "application/x-dtbncx+xml"
	case ".opf":
		return "application/oebps-package+xml"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".ttf":
		return "application/x-font-ttf"
	case ".otf":
		return "application/x-font-opentype"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}

func mediaTypeMatchesPath(mediaType, filePath string) bool {
	if mediaType == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xhtml", ".html", ".htm":
		return mediaType == "application/xhtml+xml"
	case ".css":
		return mediaType == "text/css"
	case ".ncx":
		return mediaType == "application/x-dtbncx+xml"
	case ".opf":
		return mediaType == "application/oebps-package+xml"
	case ".jpg", ".jpeg":
		return mediaType == "image/jpeg"
	case ".png":
		return mediaType == "image/png"
	case ".gif":
		return mediaType == "image/gif"
	case ".webp":
		return mediaType == "image/webp"
	case ".svg":
		return mediaType == "image/svg+xml"
	case ".ttf":
		return mediaType == "application/x-font-ttf" || mediaType == "font/ttf"
	case ".otf":
		return mediaType == "application/x-font-opentype" || mediaType == "font/otf"
	case ".woff":
		return mediaType == "font/woff" || mediaType == "application/font-woff"
	case ".woff2":
		return mediaType == "font/woff2"
	default:
		return true
	}
}

func shouldIgnoreUnmanifested(name, opfPath string) bool {
	lower := strings.ToLower(name)
	return name == "mimetype" ||
		name == opfPath ||
		lower == "meta-inf/container.xml" ||
		strings.HasPrefix(lower, "meta-inf/")
}

func isHTMLDocument(p string) bool {
	lower := strings.ToLower(p)
	return strings.HasSuffix(lower, ".xhtml") || strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm")
}

func isImageManifestItemForRepair(item ManifestItem) bool {
	return strings.HasPrefix(strings.ToLower(item.MediaType), "image/") ||
		strings.HasPrefix(strings.ToLower(contentTypeFor(item.FullPath)), "image/")
}

func isCoverPageItem(item ManifestItem) bool {
	if !isHTMLDocument(item.FullPath) || hasPropertyToken(item.Attrs["properties"], "nav") {
		return false
	}
	lower := strings.ToLower(item.ID + " " + item.Href + " " + item.FullPath)
	base := strings.ToLower(filepath.Base(item.FullPath))
	return strings.Contains(lower, "titlepage") ||
		strings.Contains(lower, "cover") ||
		base == "title.xhtml" ||
		base == "title.html"
}

func isVisibleTOCPage(item ManifestItem) bool {
	if !isHTMLDocument(item.FullPath) || hasPropertyToken(item.Attrs["properties"], "nav") {
		return false
	}
	if isCoverPageItem(item) {
		return false
	}
	base := strings.ToLower(filepath.Base(item.FullPath))
	return base == "index.html" || base == "index.xhtml" || base == "toc.html" || base == "toc.xhtml"
}

// RepairFixForIssue maps EPUB error codes to exact EPUB-Forge fix action IDs.
func RepairFixForIssue(code string) (string, bool) {
	switch code {
	case "MIMETYPE_FIRST", "MIMETYPE_MISSING", "MIMETYPE_COMPRESSED", "MIMETYPE_VALUE", "MIMETYPE_NOT_FIRST", "MIMETYPE_INVALID_VALUE":
		return "PACKAGE_MIMETYPE", true
	case "OPF_VERSION_LEGACY", "METADATA_MODIFIED_MISSING", "NAV_MISSING":
		return "UPGRADE_EPUB3", true
	case "MANIFEST_FILE_MISSING", "SPINE_IDREF_MISSING", "MANIFEST_ORPHAN_DOCUMENT", "MANIFEST_ID_DUPLICATE", "MANIFEST_HREF_DUPLICATE":
		return "REMOVE_MISSING_MANIFEST_ITEMS", true
	case "MANIFEST_MEDIA_TYPE_PARAMETER", "MANIFEST_MEDIA_TYPE_MISMATCH":
		return "FIX_MEDIA_TYPES", true
	case "ZIP_FILE_UNMANIFESTED", "LINK_TARGET_UNMANIFESTED":
		return "ADD_UNMANIFESTED_FILES", true
	case "XHTML_XML", "XHTML_NAMESPACE":
		return "FIX_XHTML", true
	case "NCX_XML", "NCX_LINK_MISSING", "NCX_IDREF_MISSING", "NCX_DUMMY_DUPLICATE_LINK", "NCX_TARGET_NOT_IN_SPINE", "TOC_NAV_NCX_MISMATCH":
		return "FIX_TOC_NCX", true
	case "VISIBLE_TOC_PAGE_MISSING":
		return "BUILD_TOC_PAGE", true
	case "COVER_PAGE_MISSING":
		return "BUILD_COVER_PAGE", true
	case "LINK_TARGET_MISSING":
		return "CLEAN_BROKEN_CONTENT_LINKS", true
	case "CHAPTER_TITLE_MISSING":
		return "BUILD_CHAPTER_TITLES", true
	default:
		return "", false
	}
}

// LoadBookContext loads and parses an EPUB archive into a rich BookContext.
func LoadBookContext(epubPath string) (*BookContext, error) {
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		return nil, err
	}

	fi, _ := os.Stat(epubPath)
	var size int64
	if fi != nil {
		size = fi.Size()
	}

	ctx := &BookContext{
		FilePath:       epubPath,
		FileName:       filepath.Base(epubPath),
		Size:           size,
		Reader:         r,
		Entries:        make(map[string]*zip.File),
		ManifestByID:   make(map[string]ManifestItem),
		ManifestByPath: make(map[string]ManifestItem),
	}

	for _, f := range r.File {
		ctx.Entries[f.Name] = f
	}

	// Read container.xml
	if cFile := ctx.Entries["META-INF/container.xml"]; cFile != nil {
		if cText, err := ctx.readText("META-INF/container.xml"); err == nil {
			reRoot := regexp.MustCompile(`(?is)<rootfile\b[^>]*\bfull-path\s*=\s*["']([^"']+)["']`)
			if m := reRoot.FindStringSubmatch(cText); len(m) == 2 {
				ctx.OPFPath = normalizeZipPath(m[1])
				ctx.OPFDir = posixDir(ctx.OPFPath)
			}
		}
	}

	// Read OPF
	if ctx.OPFPath != "" {
		if opfText, err := ctx.readText(ctx.OPFPath); err == nil {
			ctx.OPFXML = opfText

			// Title, Creator, Language
			if m := titleTextRe.FindStringSubmatch(opfText); len(m) == 2 {
				ctx.Title = cleanText(stripTags(m[1]))
				ctx.Metadata.Title = ctx.Title
			}
			if m := creatorTextRe.FindStringSubmatch(opfText); len(m) == 2 {
				ctx.Creator = cleanText(stripTags(m[1]))
				ctx.Metadata.Creator = ctx.Creator
			}
			if m := langTextRe.FindStringSubmatch(opfText); len(m) == 2 {
				ctx.Metadata.Language = strings.TrimSpace(m[1])
			}

			// Parse Manifest
			reItem := regexp.MustCompile(`(?is)<item\b([^>]*)>`)
			for _, m := range reItem.FindAllStringSubmatch(opfText, -1) {
				attrs := parseAttrs(m[0])
				id := attrs["id"]
				href := attrs["href"]
				mediaType := attrs["media-type"]
				fullPath := resolveZipHref(ctx.OPFDir, href)

				item := ManifestItem{
					ID:        id,
					Href:      href,
					FullPath:  fullPath,
					MediaType: mediaType,
					Raw:       m[0],
					Attrs:     attrs,
				}
				ctx.Manifest = append(ctx.Manifest, item)
				if id != "" {
					ctx.ManifestByID[id] = item
				}
				if fullPath != "" {
					ctx.ManifestByPath[fullPath] = item
				}

				if id == "ncx" || mediaType == "application/x-dtbncx+xml" || strings.HasSuffix(strings.ToLower(fullPath), ".ncx") {
					copyItem := item
					ctx.NCX = &copyItem
				}
			}

			// Parse Spine
			reItemref := regexp.MustCompile(`(?is)<itemref\b([^>]*)>`)
			for _, m := range reItemref.FindAllStringSubmatch(opfText, -1) {
				attrs := parseAttrs(m[0])
				idref := attrs["idref"]
				linear := strings.ToLower(attrs["linear"]) != "no"

				ref := SpineRef{
					IDRef:  idref,
					Linear: linear,
					Raw:    m[0],
					Attrs:  attrs,
				}
				ctx.Spine = append(ctx.Spine, ref)

				if item, ok := ctx.ManifestByID[idref]; ok {
					ctx.Chapters = append(ctx.Chapters, ChapterInfo{
						IDRef: idref,
						Href:  item.Href,
						Path:  item.FullPath,
						Title: ctx.htmlTitle(item.FullPath),
						Index: len(ctx.Chapters),
					})
				}
			}
		}
	}

	// Parse TOC NCX
	if ctx.NCX != nil {
		if ncxText, err := ctx.readText(ctx.NCX.FullPath); err == nil {
			ctx.TOC = parseNCX(ncxText, posixDir(ctx.NCX.FullPath))
		}
	}

	return ctx, nil
}

func (ctx *BookContext) htmlTitle(filePath string) string {
	text, err := ctx.readText(filePath)
	if err != nil {
		return ""
	}
	reH := regexp.MustCompile(`(?is)<h[1-6]\b[^>]*>(.*?)</h[1-6]>`)
	if m := reH.FindStringSubmatch(text); len(m) == 2 {
		return cleanText(stripTags(m[1]))
	}
	reTitle := regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title>`)
	if m := reTitle.FindStringSubmatch(text); len(m) == 2 {
		return cleanText(stripTags(m[1]))
	}
	return ""
}

func (ctx *BookContext) spineUsesID(id string) bool {
	for _, ref := range ctx.Spine {
		if ref.IDRef == id {
			return true
		}
	}
	return false
}

func (ctx *BookContext) findCoverImageItem() *ManifestItem {
	for _, item := range ctx.Manifest {
		if !isImageManifestItemForRepair(item) || ctx.Entries[item.FullPath] == nil {
			continue
		}
		copyItem := item
		lower := strings.ToLower(item.ID + " " + item.Href + " " + item.FullPath)
		if strings.Contains(lower, "cover") || strings.Contains(lower, "title") || strings.Contains(lower, "front") {
			return &copyItem
		}
	}
	for _, item := range ctx.Manifest {
		if isImageManifestItemForRepair(item) && ctx.Entries[item.FullPath] != nil {
			copyItem := item
			return &copyItem
		}
	}
	return nil
}

func (ctx *BookContext) findCoverPageItem() (*ManifestItem, bool) {
	for _, item := range ctx.Manifest {
		if isCoverPageItem(item) {
			copyItem := item
			return &copyItem, true
		}
	}
	return nil, false
}

func (ctx *BookContext) findVisibleTOCItem() (*ManifestItem, bool) {
	for _, item := range ctx.Manifest {
		if isVisibleTOCPage(item) {
			copyItem := item
			return &copyItem, true
		}
	}
	return nil, false
}

func (ctx *BookContext) hasNavDocument() bool {
	for _, item := range ctx.Manifest {
		if hasPropertyToken(item.Attrs["properties"], "nav") {
			return true
		}
	}
	return false
}

func (ctx *BookContext) findNavDocumentItem() (*ManifestItem, bool) {
	for _, item := range ctx.Manifest {
		if hasPropertyToken(item.Attrs["properties"], "nav") {
			copyItem := item
			return &copyItem, true
		}
	}
	for _, item := range ctx.Manifest {
		lower := strings.ToLower(item.FullPath)
		if strings.HasSuffix(lower, "nav.xhtml") || strings.HasSuffix(lower, "nav.html") {
			copyItem := item
			return &copyItem, true
		}
	}
	return nil, false
}

func (ctx *BookContext) repairPageDir() string {
	if len(ctx.Chapters) > 0 {
		if dir := posixDir(ctx.Chapters[0].Path); dir != "" {
			return dir
		}
	}
	return ctx.OPFDir
}

func parseNCX(ncxText, ncxDir string) []TocPoint {
	var points []TocPoint
	rePoint := regexp.MustCompile(`(?is)<navPoint\b[^>]*>(.*?)</navPoint>`)
	reLabel := regexp.MustCompile(`(?is)<text\b[^>]*>(.*?)</text>`)
	reSrc := regexp.MustCompile(`(?is)<content\b[^>]*\bsrc\s*=\s*["']([^"']+)["']`)

	for _, m := range rePoint.FindAllStringSubmatch(ncxText, -1) {
		block := m[1]
		title := ""
		if lm := reLabel.FindStringSubmatch(block); len(lm) == 2 {
			title = cleanText(stripTags(lm[1]))
		}
		src := ""
		if sm := reSrc.FindStringSubmatch(block); len(sm) == 2 {
			src = sm[1]
		}
		if src != "" {
			fullPath := resolveZipHref(ncxDir, src)
			points = append(points, TocPoint{
				Title:    title,
				Src:      src,
				FullPath: fullPath,
			})
		}
	}
	return points
}

func parseNavTOCPoints(navHTML, navDir string) []TocPoint {
	var points []TocPoint
	reA := regexp.MustCompile(`(?is)<a\b[^>]*\bhref\s*=\s*["']([^"']*)["'][^>]*>(.*?)</a>`)
	for _, m := range reA.FindAllStringSubmatch(navHTML, -1) {
		if len(m) >= 3 {
			href := m[1]
			title := cleanText(stripTags(m[2]))
			cleanHref := strings.Split(href, "#")[0]
			points = append(points, TocPoint{
				Title:    title,
				Src:      href,
				FullPath: resolveZipHref(navDir, cleanHref),
			})
		}
	}
	return points
}

// ValidateEPUB inspects an EPUB file and generates a comprehensive diagnosis report matching EPUB-Forge.
func ValidateEPUB(epubPath string) (ValidationReport, error) {
	ctx, err := LoadBookContext(epubPath)
	if err != nil {
		return ValidationReport{
			Valid:  false,
			Errors: 1,
			Issues: []ValidationIssue{
				{
					Severity: "error",
					Code:     "ZIP_CORRUPT",
					Message:  fmt.Sprintf("Cannot open EPUB archive: %v", err),
					Fixable:  false,
				},
			},
		}, nil
	}
	defer ctx.Close()

	var report ValidationReport
	addIssue := func(severity, code, file, message string) {
		fixID, fixable := RepairFixForIssue(code)
		report.Issues = append(report.Issues, ValidationIssue{
			Severity: severity,
			Code:     code,
			File:     file,
			Message:  message,
			Fixable:  fixable,
			FixID:    fixID,
		})
		switch severity {
		case "error":
			report.Errors++
		case "warning":
			report.Warnings++
		default:
			report.Infos++
		}
	}

	// 1. Validate Package & Mimetype
	if len(ctx.Reader.File) == 0 {
		addIssue("error", "ZIP_EMPTY", "", "EPUB archive is empty")
		report.Valid = false
		return report, nil
	}

	first := ctx.Reader.File[0]
	if first.Name != "mimetype" {
		addIssue("error", "MIMETYPE_FIRST", "mimetype", "mimetype must be the first ZIP entry")
	}
	if f := ctx.Entries["mimetype"]; f == nil {
		addIssue("error", "MIMETYPE_MISSING", "mimetype", "missing mimetype file")
	} else {
		if f.Method != zip.Store {
			addIssue("error", "MIMETYPE_COMPRESSED", "mimetype", "mimetype must be stored without compression")
		}
		data, err := ctx.readText("mimetype")
		if err != nil {
			addIssue("error", "MIMETYPE_READ", "mimetype", err.Error())
		} else if strings.TrimSpace(data) != "application/epub+zip" {
			addIssue("error", "MIMETYPE_VALUE", "mimetype", "mimetype must be exactly application/epub+zip")
		}
	}

	if ctx.Entries["META-INF/container.xml"] == nil {
		addIssue("error", "CONTAINER_MISSING", "META-INF/container.xml", "missing EPUB container.xml")
	}
	if ctx.OPFPath == "" {
		addIssue("error", "OPF_ROOT_MISSING", "META-INF/container.xml", "container.xml does not declare an OPF rootfile")
	} else if ctx.Entries[ctx.OPFPath] == nil {
		addIssue("error", "OPF_FILE_MISSING", ctx.OPFPath, "OPF rootfile does not exist in the archive")
	}

	// 2. Validate OPF
	if ctx.OPFXML != "" {
		if err := validateXMLWellFormed(ctx.OPFXML); err != nil {
			addIssue("error", "OPF_XML", ctx.OPFPath, fmt.Sprintf("OPF is not well-formed XML: %v", err))
		}

		packageTag := regexp.MustCompile(`(?is)<package\b[^>]*>`).FindString(ctx.OPFXML)
		attrs := parseAttrs(packageTag)
		version := strings.TrimSpace(attrs["version"])
		if version == "" {
			addIssue("error", "OPF_VERSION_MISSING", ctx.OPFPath, "OPF package version is missing")
		} else if version != "3.0" {
			addIssue("warning", "OPF_VERSION_LEGACY", ctx.OPFPath, "Declared EPUB version is "+version)
		}
		if strings.TrimSpace(attrs["unique-identifier"]) == "" {
			addIssue("error", "OPF_UID_MISSING", ctx.OPFPath, "OPF unique-identifier attribute is missing")
		}
		if strings.TrimSpace(ctx.Metadata.Title) == "" {
			addIssue("error", "METADATA_TITLE_MISSING", ctx.OPFPath, "dc:title is required")
		}
		if strings.TrimSpace(ctx.Metadata.Language) == "" {
			addIssue("error", "METADATA_LANGUAGE_MISSING", ctx.OPFPath, "dc:language is required")
		}
		if !regexp.MustCompile(`(?is)<dc:identifier\b[^>]*>`).MatchString(ctx.OPFXML) {
			addIssue("error", "METADATA_IDENTIFIER_MISSING", ctx.OPFPath, "dc:identifier is required")
		}
		if version == "3.0" && !regexp.MustCompile(`(?is)<meta\b[^>]*property\s*=\s*["']dcterms:modified["'][^>]*>`).MatchString(ctx.OPFXML) {
			addIssue("error", "METADATA_MODIFIED_MISSING", ctx.OPFPath, "EPUB 3 requires dcterms:modified metadata")
		}
	}

	// 3. Validate Manifest & Spine
	ids := map[string]bool{}
	paths := map[string]bool{}
	for _, item := range ctx.Manifest {
		if strings.TrimSpace(item.ID) == "" {
			addIssue("error", "MANIFEST_ID_MISSING", ctx.OPFPath, "manifest item is missing id")
		}
		if ids[item.ID] {
			addIssue("error", "MANIFEST_ID_DUPLICATE", ctx.OPFPath, "duplicate manifest id: "+item.ID)
		}
		ids[item.ID] = true

		if strings.TrimSpace(item.Href) == "" {
			addIssue("error", "MANIFEST_HREF_MISSING", ctx.OPFPath, "manifest item "+item.ID+" is missing href")
			continue
		}
		if paths[item.FullPath] {
			addIssue("warning", "MANIFEST_HREF_DUPLICATE", item.FullPath, "duplicate manifest href: "+item.Href)
		}
		paths[item.FullPath] = true

		if ctx.Entries[item.FullPath] == nil {
			addIssue("error", "MANIFEST_FILE_MISSING", item.FullPath, "manifest item points to a missing file")
		}
		if strings.Contains(item.MediaType, ";") {
			addIssue("warning", "MANIFEST_MEDIA_TYPE_PARAMETER", ctx.OPFPath, "manifest media-type should not include parameters: "+item.MediaType)
		}
		if !mediaTypeMatchesPath(item.MediaType, item.FullPath) {
			addIssue("warning", "MANIFEST_MEDIA_TYPE_MISMATCH", item.FullPath, "media-type "+item.MediaType+" does not match file extension")
		}
	}

	if len(ctx.Spine) == 0 {
		addIssue("error", "SPINE_EMPTY", ctx.OPFPath, "spine must contain at least one itemref")
	}
	spinePathSet := map[string]bool{}
	for _, ref := range ctx.Spine {
		item, ok := ctx.ManifestByID[ref.IDRef]
		if !ok {
			addIssue("error", "SPINE_IDREF_MISSING", ctx.OPFPath, "spine itemref points to missing manifest id: "+ref.IDRef)
			continue
		}
		spinePathSet[item.FullPath] = true
	}

	for _, item := range ctx.Manifest {
		if isHTMLDocument(item.FullPath) {
			if !spinePathSet[item.FullPath] && !hasPropertyToken(item.Attrs["properties"], "nav") {
				addIssue("warning", "MANIFEST_ORPHAN_DOCUMENT", item.FullPath, "File HTML mồ côi: được khai báo trong manifest nhưng không nằm trong spine")
			}
		}
	}

	for name := range ctx.Entries {
		if shouldIgnoreUnmanifested(name, ctx.OPFPath) || paths[name] {
			continue
		}
		addIssue("info", "ZIP_FILE_UNMANIFESTED", name, "file exists in ZIP but is not declared in the manifest")
	}

	// 4. Validate Navigation
	if ctx.NCX != nil {
		if ncxText, err := ctx.readText(ctx.NCX.FullPath); err == nil {
			if err := validateXMLWellFormed(ncxText); err != nil {
				addIssue("error", "NCX_XML", ctx.NCX.FullPath, fmt.Sprintf("NCX is not well-formed XML: %v", err))
			}
			ncxPoints := parseNCX(ncxText, posixDir(ctx.NCX.FullPath))
			indexCount := 0
			tocItem, _ := ctx.findVisibleTOCItem()
			for _, pt := range ncxPoints {
				if pt.FullPath == "" || ctx.Entries[pt.FullPath] == nil {
					addIssue("error", "NCX_LINK_MISSING", ctx.NCX.FullPath, "NCX points to missing file: "+pt.Src)
				} else if isHTMLDocument(pt.FullPath) && len(spinePathSet) > 0 && !spinePathSet[pt.FullPath] {
					addIssue("warning", "NCX_TARGET_NOT_IN_SPINE", ctx.NCX.FullPath, "Mục lục NCX trỏ tới file không nằm trong spine: "+pt.Src)
				}
				if tocItem != nil && pt.FullPath == tocItem.FullPath {
					indexCount++
				}
			}
			if indexCount > 1 {
				addIssue("warning", "NCX_DUMMY_DUPLICATE_LINK", ctx.NCX.FullPath, fmt.Sprintf("Mục lục NCX chứa %d mục cùng trỏ về trang mục lục HTML", indexCount))
			}
		}
	}

	// 5. Validate Content Documents
	for _, item := range ctx.Manifest {
		if !isHTMLDocument(item.FullPath) {
			continue
		}
		text, err := ctx.readText(item.FullPath)
		if err != nil {
			continue
		}
		if err := validateXMLWellFormed(text); err != nil {
			addIssue("error", "XHTML_XML", item.FullPath, fmt.Sprintf("content document is not well-formed XML: %v", err))
		}
		if !regexp.MustCompile(`(?is)<html\b[^>]*xmlns\s*=\s*["']http://www\.w3\.org/1999/xhtml["']`).MatchString(text) {
			addIssue("warning", "XHTML_NAMESPACE", item.FullPath, "XHTML document should declare the XHTML namespace")
		}

		for _, m := range hrefAttrRe.FindAllStringSubmatch(text, -1) {
			if len(m) == 2 {
				ref := m[1]
				if skipLocalReference(ref) {
					continue
				}
				resolved := resolveZipHref(posixDir(item.FullPath), ref)
				if resolved == "" {
					continue
				}
				if ctx.Entries[resolved] == nil {
					addIssue("error", "LINK_TARGET_MISSING", item.FullPath, "local reference points to missing file: "+ref)
				}
			}
		}

		if isChapterTitleMissing(text) && !isCoverPageItem(item) && !isVisibleTOCPage(item) {
			addIssue("warning", "CHAPTER_TITLE_MISSING", item.FullPath, "Chương không có tiêu đề heading (h1-h6) hiển thị")
		}
	}

	report.Valid = report.Errors == 0
	if report.Issues == nil {
		report.Issues = []ValidationIssue{}
	}
	return report, nil
}

// RepairEPUB executes the full EPUB-Forge repair suite on an EPUB archive.
func RepairEPUB(srcPath, dstPath string, opts RepairOptions) (RepairResult, error) {
	ctx, err := LoadBookContext(srcPath)
	if err != nil {
		return RepairResult{Success: false, Logs: []string{fmt.Sprintf("Failed to load EPUB: %v", err)}}, err
	}
	defer ctx.Close()

	selected := opts.toFixSelection()
	if selected.empty() {
		return RepairResult{
			Success: true,
			Logs:    []string{"Chưa chọn mục nào để sửa."},
			Report:  ValidationReport{Valid: true},
		}, nil
	}

	var logs []string
	editedFiles := make(map[string][]byte)
	removedManifestIDs := make(map[string]bool)
	correctedMediaTypes := make(map[string]string)
	var addedManifestItems []ManifestItem

	// Track duplicate manifest entries for dedup (same id appearing multiple times)
	duplicateManifestIndices := make(map[int]bool) // Indices in ctx.Manifest to skip
	if selected.has("REMOVE_MISSING_MANIFEST_ITEMS") || selected.has("FIX_MEDIA_TYPES") || selected.has("ADD_UNMANIFESTED_FILES") {
		seenIDs := make(map[string]bool) // id -> already seen
		for i, item := range ctx.Manifest {
			if item.ID == "" {
				continue
			}
			if seenIDs[item.ID] {
				// Duplicate ID detected — mark this index for removal
				// Keep the first occurrence, remove subsequent ones
				duplicateManifestIndices[i] = true
				logs = append(logs, fmt.Sprintf("[Manifest] Đã loại bỏ mục trùng lặp trong manifest: %s (ID: %s)", item.Href, item.ID))
			} else {
				seenIDs[item.ID] = true
			}
		}
	}

	// 1. Reconcile Manifest items
	for i, item := range ctx.Manifest {
		if duplicateManifestIndices[i] {
			continue // Already handled as duplicate
		}
		f, ok := ctx.Entries[item.FullPath]
		if !ok || (f.FileInfo() != nil && f.FileInfo().IsDir()) {
			if selected.has("REMOVE_MISSING_MANIFEST_ITEMS") {
				removedManifestIDs[item.ID] = true
				if ctx.spineUsesID(item.ID) {
					logs = append(logs, fmt.Sprintf("[Spine] Đã loại bỏ chương bị thiếu khỏi spine: %s (ID: %s)", item.Href, item.ID))
				} else {
					logs = append(logs, fmt.Sprintf("[Manifest] Đã loại bỏ tài nguyên bị thiếu khỏi manifest: %s (ID: %s)", item.Href, item.ID))
				}
			}
			continue
		}

		if isHTMLDocument(item.FullPath) && !ctx.spineUsesID(item.ID) && !hasPropertyToken(item.Attrs["properties"], "nav") {
			if selected.has("REMOVE_MISSING_MANIFEST_ITEMS") {
				removedManifestIDs[item.ID] = true
				logs = append(logs, fmt.Sprintf("[Manifest] Đã loại bỏ file HTML mồ côi khỏi manifest: %s (ID: %s)", item.Href, item.ID))
				continue
			}
		}

		if selected.has("FIX_MEDIA_TYPES") {
			correctMediaType := contentTypeFor(item.FullPath)
			if item.MediaType != correctMediaType {
				correctedMediaTypes[item.ID] = correctMediaType
				logs = append(logs, fmt.Sprintf("[Manifest] Đã sửa media-type cho %s: %s -> %s", item.Href, item.MediaType, correctMediaType))
			}
		}
	}

	// 2. Discover Unmanifested Files
	if selected.has("ADD_UNMANIFESTED_FILES") {
		for p, f := range ctx.Entries {
			if (f.FileInfo() != nil && f.FileInfo().IsDir()) || shouldIgnoreUnmanifested(p, ctx.OPFPath) {
				continue
			}
			lowerPath := strings.ToLower(p)
			if strings.HasSuffix(lowerPath, ".tmp") || strings.HasSuffix(lowerPath, ".bak") || strings.HasSuffix(lowerPath, ".opf") || strings.HasSuffix(lowerPath, ".ncx") {
				continue
			}
			if _, ok := ctx.ManifestByPath[p]; ok {
				continue
			}
			newID := uniqueRepairManifestID(ctx, addedManifestItems, "added_"+strings.ReplaceAll(filepath.Base(p), ".", "_"))
			addedManifestItems = append(addedManifestItems, ManifestItem{
				ID:        newID,
				Href:      relativeZipPath(ctx.OPFPath, p),
				FullPath:  p,
				MediaType: contentTypeFor(p),
			})
			logs = append(logs, fmt.Sprintf("[Manifest] Đã khai báo file chưa có trong manifest: %s (ID: %s)", p, newID))
		}
	}

	// 3. Reconcile Spine
	validSpineRefs := make([]SpineRef, 0, len(ctx.Spine))
	for _, ref := range ctx.Spine {
		item, existsInManifest := ctx.ManifestByID[ref.IDRef]
		if !existsInManifest {
			if selected.has("REMOVE_MISSING_MANIFEST_ITEMS") {
				logs = append(logs, fmt.Sprintf("[Spine] Đã loại bỏ itemref trỏ tới manifest ID không tồn tại: %s", ref.IDRef))
				continue
			}
			validSpineRefs = append(validSpineRefs, ref)
			continue
		}
		if removedManifestIDs[item.ID] {
			logs = append(logs, fmt.Sprintf("[Spine] Đã loại bỏ itemref trỏ tới file không tồn tại: %s", item.Href))
			continue
		}
		validSpineRefs = append(validSpineRefs, ref)
	}

	// 4. Content Documents (XML repair & Link cleaning)
	if selected.has("FIX_XHTML") || selected.has("CLEAN_BROKEN_CONTENT_LINKS") {
		repairContentDocuments(ctx, selected, editedFiles, removedManifestIDs, &logs)
	}

	// 5. NCX & Navigation Repair
	ncxPath := ""
	if ctx.NCX != nil {
		ncxPath = ctx.NCX.FullPath
	}
	if selected.has("FIX_TOC_NCX") {
		repairNCX(ctx, validSpineRefs, editedFiles, removedManifestIDs, &logs)
		if ctx.NCX != nil {
			ncxPath = ctx.NCX.FullPath
		} else {
			ncxPath = resolveZipHref(ctx.OPFDir, "toc.ncx")
		}
	}

	// 6. OPF Upgrades & Rebuilding
	opfContent := ctx.OPFXML
	opfChanged := false

	if (selected.has("FIX_METADATA") || opts.FixMetadata) && opfContent != "" {
		packageTag := regexp.MustCompile(`(?is)<package\b[^>]*>`).FindString(opfContent)
		if packageTag != "" {
			attrs := parseAttrs(packageTag)
			if strings.TrimSpace(attrs["unique-identifier"]) == "" {
				newPackageTag := strings.TrimSuffix(packageTag, ">") + ` unique-identifier="uid">`
				opfContent = strings.Replace(opfContent, packageTag, newPackageTag, 1)
				opfChanged = true
				logs = append(logs, "[OPF] Đã bổ sung thuộc tính unique-identifier=\"uid\".")
			}
		}

		if !regexp.MustCompile(`(?is)<dc:language\b[^>]*>`).MatchString(opfContent) {
			m := metadataRe.FindStringSubmatchIndex(opfContent)
			if len(m) >= 6 {
				langTag := "    <dc:language>en</dc:language>\n"
				opfContent = opfContent[:m[4]] + "\n" + langTag + opfContent[m[4]:]
				opfChanged = true
				logs = append(logs, "[Metadata] Đã bổ sung thẻ dc:language=en.")
			}
		}
		if !regexp.MustCompile(`(?is)<dc:identifier\b[^>]*>`).MatchString(opfContent) {
			m := metadataRe.FindStringSubmatchIndex(opfContent)
			if len(m) >= 6 {
				uidTag := fmt.Sprintf("    <dc:identifier id=\"uid\">urn:uuid:%s</dc:identifier>\n", randomID())
				opfContent = opfContent[:m[4]] + "\n" + uidTag + opfContent[m[4]:]
				opfChanged = true
				logs = append(logs, "[Metadata] Đã bổ sung thẻ dc:identifier.")
			}
		}
		if !regexp.MustCompile(`(?is)<dc:title\b[^>]*>`).MatchString(opfContent) {
			m := metadataRe.FindStringSubmatchIndex(opfContent)
			if len(m) >= 6 {
				titleTag := "    <dc:title>Untitled</dc:title>\n"
				opfContent = opfContent[:m[4]] + "\n" + titleTag + opfContent[m[4]:]
				opfChanged = true
				logs = append(logs, "[Metadata] Đã bổ sung thẻ dc:title.")
			}
		}
	}

	if selected.has("UPGRADE_EPUB3") && opfContent != "" {
		nextOPF := ensureOPFVersion3(opfContent)
		if nextOPF != opfContent {
			opfContent = nextOPF
			opfChanged = true
			logs = append(logs, "[OPF] Đã nâng package version lên EPUB 3.0.")
		}

		nextOPF = ensureDCTermsModifiedLocal(opfContent)
		if nextOPF != opfContent {
			opfContent = nextOPF
			opfChanged = true
			logs = append(logs, "[OPF] Đã bổ sung metadata dcterms:modified.")
		}

		if !ctx.hasNavDocument() {
			navPath := uniqueRepairZipPath(ctx, editedFiles, resolveZipHref(ctx.OPFDir, "nav.xhtml"))
			navID := uniqueRepairManifestID(ctx, addedManifestItems, "nav")
			addedManifestItems = append(addedManifestItems, ManifestItem{
				ID:        navID,
				Href:      relativeZipPath(ctx.OPFPath, navPath),
				FullPath:  navPath,
				MediaType: "application/xhtml+xml",
				Attrs:     map[string]string{"properties": "nav"},
			})
			editedFiles[navPath] = []byte(buildNavDocument(navPath, validSpineRefs, ctx))
			opfChanged = true
			logs = append(logs, "[NAV] Đã tạo nav.xhtml cho EPUB 3.")
		}
	}

	if selected.has("BUILD_COVER_PAGE") {
		if ensureCoverPage(ctx, editedFiles, &addedManifestItems, &validSpineRefs, &logs) {
			opfChanged = true
		}
	}

	if selected.has("BUILD_TOC_PAGE") {
		if ensureVisibleTOCPage(ctx, editedFiles, &addedManifestItems, &validSpineRefs, &logs) {
			opfChanged = true
		}
	}

	if selected.has("BUILD_CHAPTER_TITLES") {
		repairChapterTitles(ctx, editedFiles, &logs)
	}

	if selected.has("FIX_TOC_NCX") && ncxPath != "" && opfContent != "" {
		hasNCXDecl := false
		for _, item := range ctx.Manifest {
			if item.ID == "ncx" || item.MediaType == "application/x-dtbncx+xml" || strings.HasSuffix(strings.ToLower(item.FullPath), ".ncx") {
				hasNCXDecl = true
				break
			}
		}
		if !hasNCXDecl {
			addedManifestItems = append(addedManifestItems, ManifestItem{
				ID:        uniqueRepairManifestID(ctx, addedManifestItems, "ncx"),
				Href:      relativeZipPath(ctx.OPFPath, ncxPath),
				FullPath:  ncxPath,
				MediaType: "application/x-dtbncx+xml",
			})
			opfChanged = true
			logs = append(logs, "[Mục lục] Đã khai báo NCX trong manifest.")
		}
		spineStartMatch := regexp.MustCompile(`(?i)<spine\b[^>]*>`).FindString(opfContent)
		if spineStartMatch != "" {
			attrs := parseAttrs(spineStartMatch)
			if attrs["toc"] == "" {
				opfContent = strings.Replace(opfContent, spineStartMatch, `<spine toc="ncx">`, 1)
				opfChanged = true
				logs = append(logs, "[Mục lục] Đã gắn spine toc=\"ncx\".")
			}
		}
	}

	if selected.has("REMOVE_MISSING_MANIFEST_ITEMS") ||
		selected.has("FIX_MEDIA_TYPES") ||
		selected.has("ADD_UNMANIFESTED_FILES") ||
		selected.has("FIX_TOC_NCX") ||
		selected.has("UPGRADE_EPUB3") ||
		selected.has("BUILD_TOC_PAGE") ||
		selected.has("BUILD_COVER_PAGE") {
		nextOPF := rebuildOPFManifestAndSpine(ctx, opfContent, validSpineRefs, removedManifestIDs, duplicateManifestIndices, correctedMediaTypes, addedManifestItems)
		if nextOPF != opfContent {
			opfContent = nextOPF
			opfChanged = true
		}
	}

	if opfChanged && ctx.OPFPath != "" {
		editedFiles[ctx.OPFPath] = []byte(opfContent)
	}

	// 7. Write Repaired Archive
	normalizeMimetype := selected.has("PACKAGE_MIMETYPE")
	tmpOutput := dstPath + ".repair.tmp"
	out, err := os.Create(tmpOutput)
	if err != nil {
		return RepairResult{Success: false, Logs: logs}, err
	}

	bufOut := bufio.NewWriterSize(out, 2*1024*1024)
	zw := zip.NewWriter(bufOut)
	copyBuf := make([]byte, 1024*1024)

	// Write Mimetype first
	if normalizeMimetype {
		header := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
		header.SetMode(0644)
		w, err := zw.CreateHeader(header)
		if err == nil {
			_, _ = w.Write([]byte("application/epub+zip"))
			logs = append(logs, "[Mimetype] Đã chuẩn hóa mimetype ở vị trí đầu file và không nén.")
		}
	}

	removedPaths := make(map[string]bool)
	for _, item := range ctx.Manifest {
		if removedManifestIDs[item.ID] {
			removedPaths[item.FullPath] = true
		}
	}

	written := map[string]bool{}
	if normalizeMimetype {
		written["mimetype"] = true
	}

	for _, f := range ctx.Reader.File {
		if (f.FileInfo() != nil && f.FileInfo().IsDir()) || written[f.Name] || removedPaths[f.Name] {
			continue
		}

		if content, ok := editedFiles[f.Name]; ok {
			header := &zip.FileHeader{Name: f.Name, Method: zip.Deflate}
			header.SetMode(f.Mode())
			writer, err := zw.CreateHeader(header)
			if err == nil {
				_, _ = writer.Write(content)
			}
		} else {
			_ = copyZipEntry(zw, f, copyBuf)
		}
		written[f.Name] = true
	}

	for path, content := range editedFiles {
		if written[path] {
			continue
		}
		header := &zip.FileHeader{Name: path, Method: zip.Deflate}
		header.SetMode(0644)
		writer, err := zw.CreateHeader(header)
		if err == nil {
			_, _ = writer.Write(content)
		}
		written[path] = true
	}

	_ = zw.Close()
	_ = bufOut.Flush()
	_ = out.Close()

	// Atomic replace
	_ = os.Rename(tmpOutput, dstPath)

	report, _ := ValidateEPUB(dstPath)
	return RepairResult{
		Success:    true,
		FixedCount: len(logs),
		Logs:       logs,
		Report:     report,
	}, nil
}

func copyZipEntry(zw *zip.Writer, f *zip.File, buf []byte) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	header := &zip.FileHeader{
		Name:   f.Name,
		Method: f.Method,
	}
	header.SetMode(f.Mode())

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.CopyBuffer(w, rc, buf)
	return err
}

func repairChapterTitles(ctx *BookContext, editedFiles map[string][]byte, logs *[]string) {
	reBody := regexp.MustCompile(`(?i)(<body[^>]*>)`)

	for _, ch := range ctx.Chapters {
		if !isHTMLDocument(ch.Path) || isCoverPageItem(ManifestItem{FullPath: ch.Path, ID: ch.IDRef}) {
			continue
		}

		var htmlStr string
		if edited, ok := editedFiles[ch.Path]; ok {
			htmlStr = string(edited)
		} else {
			text, errRead := ctx.readText(ch.Path)
			if errRead != nil {
				continue
			}
			htmlStr = text
		}

		if !isChapterTitleMissing(htmlStr) {
			continue
		}

		bodyMatch := reBody.FindStringSubmatchIndex(htmlStr)
		if len(bodyMatch) >= 2 {
			bodyTagEnd := bodyMatch[1]
			title := ch.Title
			if title == "" {
				title = fmt.Sprintf("Chương %d", ch.Index+1)
			}
			heading := fmt.Sprintf("\n  <h2>%s</h2>", escapeXML(title))
			newHtml := htmlStr[:bodyTagEnd] + heading + htmlStr[bodyTagEnd:]
			editedFiles[ch.Path] = []byte(newHtml)
			*logs = append(*logs, fmt.Sprintf("[Tiêu đề] Đã bổ sung tiêu đề <h2> vào chương: %s (%s)", title, ch.Href))
		}
	}
}

func repairContentDocuments(ctx *BookContext, selected repairSelection, editedFiles map[string][]byte, removedManifestIDs map[string]bool, logs *[]string) {
	reTags := regexp.MustCompile(`(?i)<(br|hr|img|link|meta)\b([^>]*?)>`)
	reHtml := regexp.MustCompile(`(?i)<html\b([^>]*?)>`)

	for _, item := range ctx.Manifest {
		if removedManifestIDs[item.ID] || !isHTMLDocument(item.FullPath) {
			continue
		}

		var htmlStr string
		if edited, ok := editedFiles[item.FullPath]; ok {
			htmlStr = string(edited)
		} else {
			text, errRead := ctx.readText(item.FullPath)
			if errRead != nil {
				continue
			}
			htmlStr = text
		}

		originalHTML := htmlStr
		fixedHTML := originalHTML
		xmlFixCount := 0

		if selected.has("FIX_XHTML") {
			fixedHTML = reTags.ReplaceAllStringFunc(fixedHTML, func(m string) string {
				trimmed := strings.TrimSpace(m)
				if strings.HasSuffix(trimmed, "/>") {
					return m
				}
				tagBody := strings.TrimSpace(strings.TrimSuffix(m[1:len(m)-1], "/"))
				xmlFixCount++
				return "<" + tagBody + " />"
			})

			fixedHTML = reHtml.ReplaceAllStringFunc(fixedHTML, func(m string) string {
				if strings.Contains(m, "xmlns=") {
					return m
				}
				tagBody := m[1 : len(m)-1]
				xmlFixCount++
				return "<" + tagBody + ` xmlns="http://www.w3.org/1999/xhtml">`
			})

			var entityFixCount int
			fixedHTML, entityFixCount = repairXHTMLEntities(fixedHTML)
			xmlFixCount += entityFixCount
		}

		if selected.has("CLEAN_BROKEN_CONTENT_LINKS") {
			cleanedHTML, cleanLogs := cleanBrokenAnchorHrefs(ctx, item.FullPath, fixedHTML)
			if len(cleanLogs) > 0 {
				fixedHTML = cleanedHTML
				*logs = append(*logs, cleanLogs...)
			}

			cleanedHTML, cleanLogs = cleanBrokenImages(ctx, item.FullPath, fixedHTML)
			if len(cleanLogs) > 0 {
				fixedHTML = cleanedHTML
				*logs = append(*logs, cleanLogs...)
			}
		}

		if fixedHTML != originalHTML {
			editedFiles[item.FullPath] = []byte(fixedHTML)
			if xmlFixCount > 0 {
				*logs = append(*logs, fmt.Sprintf("[XHTML] Đã sửa %d lỗi cú pháp XML trong %s", xmlFixCount, item.Href))
			}
		}
	}
}

var xhtmlNamedEntityReplacements = map[string]string{
	"nbsp":   "&#160;",
	"copy":   "&#169;",
	"reg":    "&#174;",
	"trade":  "&#8482;",
	"ndash":  "&#8211;",
	"mdash":  "&#8212;",
	"hellip": "&#8230;",
	"lsquo":  "&#8216;",
	"rsquo":  "&#8217;",
	"ldquo":  "&#8220;",
	"rdquo":  "&#8221;",
	"laquo":  "&#171;",
	"raquo":  "&#187;",
	"middot": "&#183;",
	"bull":   "&#8226;",
	"times":  "&#215;",
	"divide": "&#247;",
	"plusmn": "&#177;",
	"euro":   "&#8364;",
	"pound":  "&#163;",
	"yen":    "&#165;",
}

func repairXHTMLEntities(input string) (string, int) {
	fixCount := 0
	entityRe := regexp.MustCompile(`&(#\d+|#x[0-9a-fA-F]+|[A-Za-z][A-Za-z0-9]+);`)
	fixed := entityRe.ReplaceAllStringFunc(input, func(entity string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(entity, "&"), ";")
		lowerName := strings.ToLower(name)
		if strings.HasPrefix(lowerName, "#") {
			return entity
		}
		switch lowerName {
		case "amp", "lt", "gt", "quot", "apos":
			return entity
		}
		if replacement, ok := xhtmlNamedEntityReplacements[lowerName]; ok {
			fixCount++
			return replacement
		}
		fixCount++
		return "&amp;" + name + ";"
	})

	var bareAmpFixCount int
	fixed, bareAmpFixCount = escapeBareAmpersands(fixed)
	fixCount += bareAmpFixCount

	return fixed, fixCount
}

func escapeBareAmpersands(input string) (string, int) {
	var b strings.Builder
	fixCount := 0
	for i := 0; i < len(input); i++ {
		if input[i] == '&' && !hasValidXMLCharacterReferenceAt(input, i) {
			b.WriteString("&amp;")
			fixCount++
			continue
		}
		b.WriteByte(input[i])
	}
	if fixCount == 0 {
		return input, 0
	}
	return b.String(), fixCount
}

func hasValidXMLCharacterReferenceAt(input string, index int) bool {
	rest := input[index:]
	for _, entity := range []string{"&amp;", "&lt;", "&gt;", "&quot;", "&apos;"} {
		if strings.HasPrefix(rest, entity) {
			return true
		}
	}

	if strings.HasPrefix(rest, "&#x") || strings.HasPrefix(rest, "&#X") {
		pos := index + 3
		start := pos
		for pos < len(input) && isHexDigit(input[pos]) {
			pos++
		}
		return pos > start && pos < len(input) && input[pos] == ';'
	}
	if strings.HasPrefix(rest, "&#") {
		pos := index + 2
		start := pos
		for pos < len(input) && input[pos] >= '0' && input[pos] <= '9' {
			pos++
		}
		return pos > start && pos < len(input) && input[pos] == ';'
	}
	return false
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func cleanBrokenAnchorHrefs(ctx *BookContext, htmlPath, htmlContent string) (string, []string) {
	var logs []string
	baseDir := posixDir(htmlPath)
	reAnchor := regexp.MustCompile(`(?is)<a\b[^>]*[\s]href\s*=\s*["']([^"']*)["'][^>]*>`)
	reHref := regexp.MustCompile(`(?is)\s+href\s*=\s*["'][^"']*["']`)

	cleanedHTML := reAnchor.ReplaceAllStringFunc(htmlContent, func(anchor string) string {
		m := reAnchor.FindStringSubmatch(anchor)
		if len(m) < 2 {
			return anchor
		}
		href := strings.TrimSpace(m[1])
		if skipLocalReference(href) {
			return anchor
		}
		resolved := resolveZipHref(baseDir, href)
		if resolved == "" || ctx.Entries[resolved] != nil {
			return anchor
		}
		logs = append(logs, fmt.Sprintf("[Nội dung] Đã bỏ href hỏng trong %s: %s", htmlPath, href))
		return reHref.ReplaceAllString(anchor, "")
	})

	return cleanedHTML, logs
}

func cleanBrokenImages(ctx *BookContext, htmlPath, htmlContent string) (string, []string) {
	var logs []string
	baseDir := posixDir(htmlPath)

	reImg := regexp.MustCompile(`(?is)<img\b[^>]*>`)
	reSrc := regexp.MustCompile(`(?is)\bsrc\s*=\s*["']([^"']*)["']`)

	cleanedHTML := reImg.ReplaceAllStringFunc(htmlContent, func(imgTag string) string {
		m := reSrc.FindStringSubmatch(imgTag)
		if len(m) < 2 {
			return imgTag
		}
		src := strings.TrimSpace(m[1])
		if skipLocalReference(src) {
			return imgTag
		}
		resolved := resolveZipHref(baseDir, src)
		if resolved == "" || ctx.Entries[resolved] != nil {
			return imgTag
		}
		logs = append(logs, fmt.Sprintf("[Nội dung] Đã xóa thẻ ảnh bị hỏng trong %s: %s", htmlPath, src))
		return ""
	})

	return cleanedHTML, logs
}

func repairNCX(ctx *BookContext, validSpineRefs []SpineRef, editedFiles map[string][]byte, removedManifestIDs map[string]bool, logs *[]string) {
	ncxPath := resolveZipHref(ctx.OPFDir, "toc.ncx")
	var ncxContent string
	if ctx.NCX != nil {
		ncxPath = ctx.NCX.FullPath
		if data, err := ctx.readText(ncxPath); err == nil {
			ncxContent = data
		}
	}

	if ncxContent == "" || validateXMLWellFormed(ncxContent) != nil {
		tocPoints := tocFromSpine(ctx, ncxPath, validSpineRefs)
		editedFiles[ncxPath] = []byte(rebuildNCXFromTOC(ncxPath, tocPoints, ctx.Title))
		*logs = append(*logs, "[Mục lục] Đã dựng lại file NCX mới từ danh sách chương.")
		return
	}

	spinePaths := map[string]bool{}
	for _, ref := range validSpineRefs {
		if item, ok := ctx.ManifestByID[ref.IDRef]; ok {
			spinePaths[item.FullPath] = true
		}
	}

	var updatedTOC []TocPoint
	tocFixedCount := 0

	for _, point := range ctx.TOC {
		resolved := resolveZipHref(posixDir(ncxPath), point.Src)
		if resolved == "" {
			tocFixedCount++
			continue
		}
		targetItem, ok := ctx.ManifestByPath[resolved]
		if !ok || removedManifestIDs[targetItem.ID] {
			tocFixedCount++
			*logs = append(*logs, fmt.Sprintf("[Mục lục] Đã loại bỏ liên kết hỏng: %s -> %s", point.Title, point.Src))
			continue
		}
		if isHTMLDocument(resolved) && len(spinePaths) > 0 && !spinePaths[resolved] {
			tocFixedCount++
			*logs = append(*logs, fmt.Sprintf("[Mục lục] Đã loại bỏ mục trỏ tới file không có trong spine: %s", point.Title))
			continue
		}
		updatedTOC = append(updatedTOC, point)
	}

	if tocFixedCount > 0 {
		ncxContent = rebuildNCXFromTOC(ncxPath, updatedTOC, ctx.Title)
		editedFiles[ncxPath] = []byte(ncxContent)
	}
}

func rebuildNCXFromTOC(ncxPath string, toc []TocPoint, title string) string {
	var b strings.Builder
	b.WriteString("<?xml version='1.0' encoding='utf-8'?>\n")
	b.WriteString(`<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">` + "\n")
	b.WriteString("  <head>\n")
	b.WriteString(fmt.Sprintf(`    <meta name="dtb:uid" content="%s"/>`+"\n", randomID()))
	b.WriteString(`    <meta name="dtb:depth" content="1"/>` + "\n")
	b.WriteString(`    <meta name="dtb:totalPageCount" content="0"/>` + "\n")
	b.WriteString(`    <meta name="dtb:maxPageNumber" content="0"/>` + "\n")
	b.WriteString("  </head>\n")
	b.WriteString(fmt.Sprintf("  <docTitle><text>%s</text></docTitle>\n", escapeXML(title)))
	b.WriteString("  <navMap>\n")
	for i, pt := range toc {
		b.WriteString(fmt.Sprintf(`    <navPoint id="nav-%d" playOrder="%d">`+"\n", i+1, i+1))
		b.WriteString(fmt.Sprintf("      <navLabel><text>%s</text></navLabel>\n", escapeXML(pt.Title)))
		b.WriteString(fmt.Sprintf(`      <content src="%s"/>`+"\n", escapeXML(relativeZipPath(ncxPath, pt.FullPath))))
		b.WriteString("    </navPoint>\n")
	}
	b.WriteString("  </navMap>\n</ncx>")
	return b.String()
}

func tocFromSpine(ctx *BookContext, ncxPath string, spineRefs []SpineRef) []TocPoint {
	var tocPoints []TocPoint
	for _, ref := range spineRefs {
		item, ok := ctx.ManifestByID[ref.IDRef]
		if !ok || !isHTMLDocument(item.FullPath) {
			continue
		}
		title := ctx.htmlTitle(item.FullPath)
		if title == "" {
			title = item.Href
		}
		tocPoints = append(tocPoints, TocPoint{
			Title:    title,
			Src:      relativeZipPath(ncxPath, item.FullPath),
			FullPath: item.FullPath,
		})
	}
	return tocPoints
}

func ensureCoverPage(ctx *BookContext, editedFiles map[string][]byte, addedManifestItems *[]ManifestItem, validSpineRefs *[]SpineRef, logs *[]string) bool {
	coverImage := ctx.findCoverImageItem()
	if coverImage == nil {
		return false
	}

	changed := false
	var coverPage ManifestItem
	if existing, ok := ctx.findCoverPageItem(); ok && existing != nil {
		coverPage = *existing
	} else {
		coverPath := uniqueRepairZipPath(ctx, editedFiles, normalizeZipPath(ctx.repairPageDir()+"cover.xhtml"))
		coverID := uniqueRepairManifestID(ctx, *addedManifestItems, "cover_page")
		coverPage = ManifestItem{
			ID:        coverID,
			Href:      relativeZipPath(ctx.OPFPath, coverPath),
			FullPath:  coverPath,
			MediaType: "application/xhtml+xml",
		}
		*addedManifestItems = append(*addedManifestItems, coverPage)
		changed = true
		*logs = append(*logs, fmt.Sprintf("[Cover] Đã tạo trang cover: %s", coverPath))
	}

	nextHTML := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>Cover</title>
  <style type="text/css">
    html, body { margin: 0; padding: 0; background: #ffffff; text-align: center; }
    .cover { margin: 0; padding: 0; text-align: center; }
    .cover img { max-width: 100%%; max-height: 100vh; height: auto; width: auto; }
  </style>
</head>
<body>
  <div class="cover">
    <img src="%s" alt="%s" />
  </div>
</body>
</html>`, escapeXML(relativeZipPath(coverPage.FullPath, coverImage.FullPath)), escapeXML(ctx.Title))

	editedFiles[coverPage.FullPath] = []byte(nextHTML)

	// Ensure cover is first in spine
	newSpine := make([]SpineRef, 0, len(*validSpineRefs)+1)
	newSpine = append(newSpine, SpineRef{IDRef: coverPage.ID, Linear: true})
	for _, ref := range *validSpineRefs {
		if ref.IDRef != coverPage.ID {
			newSpine = append(newSpine, ref)
		}
	}
	*validSpineRefs = newSpine
	return changed
}

func ensureVisibleTOCPage(ctx *BookContext, editedFiles map[string][]byte, addedManifestItems *[]ManifestItem, validSpineRefs *[]SpineRef, logs *[]string) bool {
	changed := false
	var tocItem ManifestItem
	if existing, ok := ctx.findVisibleTOCItem(); ok && existing != nil {
		tocItem = *existing
	} else {
		tocPath := uniqueRepairZipPath(ctx, editedFiles, normalizeZipPath(ctx.repairPageDir()+"toc.xhtml"))
		tocID := uniqueRepairManifestID(ctx, *addedManifestItems, "toc_page")
		tocItem = ManifestItem{
			ID:        tocID,
			Href:      relativeZipPath(ctx.OPFPath, tocPath),
			FullPath:  tocPath,
			MediaType: "application/xhtml+xml",
		}
		*addedManifestItems = append(*addedManifestItems, tocItem)
		changed = true
		*logs = append(*logs, fmt.Sprintf("[TOC] Đã tạo trang mục lục: %s", tocPath))
	}

	var links strings.Builder
	for _, ref := range *validSpineRefs {
		item, ok := ctx.ManifestByID[ref.IDRef]
		if !ok || !isHTMLDocument(item.FullPath) || item.FullPath == tocItem.FullPath || isCoverPageItem(item) {
			continue
		}
		title := ctx.htmlTitle(item.FullPath)
		if title == "" {
			title = item.Href
		}
		links.WriteString(fmt.Sprintf(`    <li><a href="%s">%s</a></li>`+"\n", escapeXML(relativeZipPath(tocItem.FullPath, item.FullPath)), escapeXML(title)))
	}

	tocHTML := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>%s</title>
</head>
<body>
  <h1>%s</h1>
  <nav id="toc">
    <ol>
%s    </ol>
  </nav>
</body>
</html>`, escapeXML(ctx.Title), escapeXML(ctx.Title), links.String())

	editedFiles[tocItem.FullPath] = []byte(tocHTML)

	// Ensure TOC is in spine (after cover if present, otherwise before first chapter)
	tocInSpine := false
	for _, ref := range *validSpineRefs {
		if ref.IDRef == tocItem.ID {
			tocInSpine = true
			break
		}
	}
	if !tocInSpine {
		targetIdx := 0
		if len(*validSpineRefs) > 0 && isCoverPageItem(ManifestItem{ID: (*validSpineRefs)[0].IDRef}) {
			targetIdx = 1
		}
		newSpine := make([]SpineRef, 0, len(*validSpineRefs)+1)
		newSpine = append(newSpine, (*validSpineRefs)[:targetIdx]...)
		newSpine = append(newSpine, SpineRef{IDRef: tocItem.ID, Linear: true})
		newSpine = append(newSpine, (*validSpineRefs)[targetIdx:]...)
		*validSpineRefs = newSpine
		changed = true
		*logs = append(*logs, "[Spine] Đã đưa trang mục lục vào reading order.")
	}

	return changed
}

func buildNavDocument(navPath string, spineRefs []SpineRef, ctx *BookContext) string {
	var links strings.Builder
	for _, ref := range spineRefs {
		item, ok := ctx.ManifestByID[ref.IDRef]
		if !ok || !isHTMLDocument(item.FullPath) {
			continue
		}
		title := ctx.htmlTitle(item.FullPath)
		if title == "" {
			title = item.Href
		}
		links.WriteString(fmt.Sprintf(`      <li><a href="%s">%s</a></li>`+"\n", escapeXML(relativeZipPath(navPath, item.FullPath)), escapeXML(title)))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head>
  <title>Navigation</title>
</head>
<body>
  <nav epub:type="toc" id="toc">
    <h1>Mục lục</h1>
    <ol>
%s    </ol>
  </nav>
</body>
</html>`, links.String())
}

func ensureOPFVersion3(opfContent string) string {
	rePackage := regexp.MustCompile(`(?is)<package\b[^>]*>`)
	return rePackage.ReplaceAllStringFunc(opfContent, func(tag string) string {
		reVersion := regexp.MustCompile(`(?is)\sversion\s*=\s*["'][^"']*["']`)
		if reVersion.MatchString(tag) {
			return reVersion.ReplaceAllString(tag, ` version="3.0"`)
		}
		return strings.TrimSuffix(tag, ">") + ` version="3.0">`
	})
}

func ensureDCTermsModifiedLocal(opfContent string) string {
	if regexp.MustCompile(`(?is)<meta\b[^>]*property\s*=\s*["']dcterms:modified["'][^>]*>`).MatchString(opfContent) {
		return opfContent
	}
	meta := fmt.Sprintf("    <meta property=\"dcterms:modified\">%s</meta>\n", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	m := metadataRe.FindStringSubmatchIndex(opfContent)
	if len(m) < 6 {
		return opfContent
	}
	return opfContent[:m[4]] + "\n" + meta + opfContent[m[4]:]
}

func rebuildOPFManifestAndSpine(ctx *BookContext, opfContent string, validSpineRefs []SpineRef, removedManifestIDs map[string]bool, duplicateIndices map[int]bool, correctedMediaTypes map[string]string, addedManifestItems []ManifestItem) string {
	var updatedManifestItems []string
	for i, item := range ctx.Manifest {
		if removedManifestIDs[item.ID] {
			continue
		}
		if duplicateIndices[i] {
			continue
		}
		mediaType := item.MediaType
		if correctedMediaType, ok := correctedMediaTypes[item.ID]; ok {
			mediaType = correctedMediaType
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(`<item id="%s" href="%s" media-type="%s"`, escapeXML(item.ID), escapeXML(item.Href), escapeXML(mediaType)))
		for k, v := range item.Attrs {
			if k != "id" && k != "href" && k != "media-type" {
				sb.WriteString(fmt.Sprintf(` %s="%s"`, k, escapeXML(v)))
			}
		}
		sb.WriteString(" />")
		updatedManifestItems = append(updatedManifestItems, "    "+sb.String())
	}
	for _, item := range addedManifestItems {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(`<item id="%s" href="%s" media-type="%s"`, escapeXML(item.ID), escapeXML(item.Href), escapeXML(item.MediaType)))
		for k, v := range item.Attrs {
			sb.WriteString(fmt.Sprintf(` %s="%s"`, k, escapeXML(v)))
		}
		sb.WriteString(" />")
		updatedManifestItems = append(updatedManifestItems, "    "+sb.String())
	}
	opfContent = replaceXMLBlock(manifestRe, opfContent, strings.Join(updatedManifestItems, "\n"))

	var updatedSpineRefs []string
	for _, ref := range validSpineRefs {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(`<itemref idref="%s"`, escapeXML(ref.IDRef)))
		if !ref.Linear {
			sb.WriteString(` linear="no"`)
		}
		for k, v := range ref.Attrs {
			if k != "idref" && k != "linear" {
				sb.WriteString(fmt.Sprintf(` %s="%s"`, k, escapeXML(v)))
			}
		}
		sb.WriteString(" />")
		updatedSpineRefs = append(updatedSpineRefs, "    "+sb.String())
	}
	return replaceXMLBlock(spineRe, opfContent, strings.Join(updatedSpineRefs, "\n"))
}

func uniqueRepairManifestID(ctx *BookContext, added []ManifestItem, base string) string {
	base = sanitizeManifestID(base)
	candidate := base
	for i := 1; ; i++ {
		exists := false
		if _, ok := ctx.ManifestByID[candidate]; ok {
			exists = true
		}
		for _, item := range added {
			if item.ID == candidate {
				exists = true
				break
			}
		}
		if !exists {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", base, i)
	}
}

func uniqueRepairZipPath(ctx *BookContext, editedFiles map[string][]byte, preferred string) string {
	if preferred == "" {
		preferred = "nav.xhtml"
	}
	ext := filepath.Ext(preferred)
	base := strings.TrimSuffix(preferred, ext)
	if ext == "" {
		ext = ".xhtml"
	}
	candidate := preferred
	for i := 1; ; i++ {
		if ctx.Entries[candidate] == nil && editedFiles[candidate] == nil {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
	}
}
