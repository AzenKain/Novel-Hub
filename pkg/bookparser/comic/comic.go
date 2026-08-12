package comic

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
)

type Parser struct {
	format string
}

func NewParser(format string) *Parser {
	return &Parser{format: bookparser.NormalizeFormat(format)}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{
		Title: bookparser.TitleFromPath(filePath),
	}
	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			ext := strings.ToLower(filepath.Ext(images[0]))
			if ext == ".png" {
				meta.CoverType = "image/png"
			} else if ext == ".webp" {
				meta.CoverType = "image/webp"
			} else {
				meta.CoverType = "image/jpeg"
			}
		}
	}
	for _, name := range []string{"ComicInfo.xml", "comicinfo.xml"} {
		xmlData, err := p.GetAsset(filePath, name)
		if err != nil || len(xmlData) == 0 {
			continue
		}
		xmlMeta, err := ParseComicInfoXML(bytes.NewReader(xmlData))
		if err != nil || xmlMeta == nil {
			continue
		}
		if xmlMeta.Title != "" {
			meta.Title = xmlMeta.Title
		}
		meta.Series = xmlMeta.Series
		meta.SeriesIndex = xmlMeta.SeriesIndex
		meta.Description = xmlMeta.Description
		meta.Author = xmlMeta.Author
		meta.Publisher = xmlMeta.Publisher
		meta.Language = xmlMeta.Language
		meta.Subjects = xmlMeta.Subjects
		meta.ReadingDirection = xmlMeta.ReadingDirection
		break
	}

	return bookparser.MergeMetadataSidecar(filePath, meta), nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	images, err := p.ListImages(filePath)
	if err != nil {
		return nil, err
	}
	title := bookparser.TitleFromPath(filePath)
	if len(images) > 0 {
		title = fmt.Sprintf("%s (%d pages)", title, len(images))
	}
	return []bookparser.ChapterData{{
		Title:       title,
		ContentPath: "comic",
		Index:       0,
	}}, nil
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
	images, err := p.ListImages(filePath)
	if err != nil {
		return "", err
	}
	if len(images) == 0 {
		return `<article><p>No comic pages were found in this archive.</p></article>`, nil
	}
	var out strings.Builder
	out.WriteString(`<article class="novelhub-comic" style="max-width: min(100%, 1100px); margin: 0 auto;">`)
	for index, imagePath := range images {
		out.WriteString(`<figure style="margin: 0 0 1.5rem;"><img src="`)
		out.WriteString(html.EscapeString(imagePath))
		out.WriteString(`" alt="Page `)
		out.WriteString(fmt.Sprintf("%d", index+1))
		out.WriteString(`" loading="lazy" style="display: block; width: 100%; height: auto;" /></figure>`)
	}
	out.WriteString(`</article>`)
	return out.String(), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	format := p.format
	if format == "" {
		format = bookparser.FormatFromPath(filePath)
	}
	switch format {
	case "cbz":
		return getZipAsset(filePath, assetPath)
	case "cbr", "rar":
		return getRARAsset(filePath, assetPath)
	case "cbt":
		return getTarAsset(filePath, assetPath)
	case "cb7", "7z":
		return getSevenZipAsset(filePath, assetPath)
	default:
		return nil, fmt.Errorf("comic parser does not support %q", format)
	}
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	format := p.format
	if format == "" {
		format = bookparser.FormatFromPath(filePath)
	}
	switch format {
	case "cbz":
		return listZipImages(filePath)
	case "cbr", "rar":
		return listRARImages(filePath)
	case "cbt":
		return listTarImages(filePath)
	case "cb7", "7z":
		return listSevenZipImages(filePath)
	default:
		return nil, fmt.Errorf("comic parser does not support %q", format)
	}
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func archiveBudgetAdd(entries *int, total *int64, size int64) error {
	*entries = *entries + 1
	if *entries > constants.MaxArchiveEntries {
		return fmt.Errorf("archive has more than %d entries", constants.MaxArchiveEntries)
	}
	if size < 0 || size > int64(constants.MaxArchiveAssetSize) {
		return fmt.Errorf("archive entry exceeds %d-byte limit", constants.MaxArchiveAssetSize)
	}
	if *total > constants.MaxArchiveUncompressedBytes-size {
		return fmt.Errorf("archive exceeds %d-byte uncompressed limit", constants.MaxArchiveUncompressedBytes)
	}
	*total += size
	return nil
}

func zipArchiveBudget(files []*zip.File) error {
	entries := 0
	var total int64
	for _, file := range files {
		if err := archiveBudgetAdd(&entries, &total, int64(file.UncompressedSize64)); err != nil {
			return err
		}
	}
	return nil
}

func sevenZipArchiveBudget(files []*sevenzip.File) error {
	entries := 0
	var total int64
	for _, file := range files {
		if err := archiveBudgetAdd(&entries, &total, int64(file.UncompressedSize)); err != nil {
			return err
		}
	}
	return nil
}

func listZipImages(filePath string) ([]string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbz: %w", err)
	}
	defer reader.Close()
	if err := zipArchiveBudget(reader.File); err != nil {
		return nil, fmt.Errorf("validate cbz: %w", err)
	}
	images := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !isComicImage(file.Name) {
			continue
		}
		images = append(images, filepath.ToSlash(file.Name))
	}
	sortComicPaths(images)
	return images, nil
}

func getZipAsset(filePath string, assetPath string) ([]byte, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbz: %w", err)
	}
	defer reader.Close()
	if err := zipArchiveBudget(reader.File); err != nil {
		return nil, fmt.Errorf("validate cbz: %w", err)
	}
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	for _, file := range reader.File {
		if filepath.ToSlash(file.Name) != assetPath {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open cbz asset: %w", err)
		}
		defer rc.Close()
		return bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
	}
	return nil, fmt.Errorf("cbz asset %q not found", assetPath)
}

func rarEntrySize(header *rardecode.FileHeader) (int64, error) {
	if header.UnKnownSize {
		return 0, fmt.Errorf("cbr entry %q declares no uncompressed size", header.Name)
	}
	return header.UnPackedSize, nil
}

func listRARImages(filePath string) ([]string, error) {
	reader, err := rardecode.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbr: %w", err)
	}
	defer reader.Close()
	var images []string
	entries := 0
	var total int64
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read cbr: %w", err)
		}
		size, err := rarEntrySize(header)
		if err != nil {
			return nil, fmt.Errorf("validate cbr: %w", err)
		}
		if err := archiveBudgetAdd(&entries, &total, size); err != nil {
			return nil, fmt.Errorf("validate cbr: %w", err)
		}
		if header.IsDir || !isComicImage(header.Name) {
			continue
		}
		images = append(images, filepath.ToSlash(header.Name))
	}
	sortComicPaths(images)
	return images, nil
}

func getRARAsset(filePath string, assetPath string) ([]byte, error) {
	reader, err := rardecode.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbr: %w", err)
	}
	defer reader.Close()
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	entries := 0
	var total int64
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read cbr: %w", err)
		}
		size, err := rarEntrySize(header)
		if err != nil {
			return nil, fmt.Errorf("validate cbr: %w", err)
		}
		if err := archiveBudgetAdd(&entries, &total, size); err != nil {
			return nil, fmt.Errorf("validate cbr: %w", err)
		}
		if filepath.ToSlash(header.Name) != assetPath {
			continue
		}
		return bookparser.ReadAllLimit(reader, constants.MaxArchiveAssetSize)
	}
	return nil, fmt.Errorf("cbr asset %q not found", assetPath)
}

func listTarImages(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbt: %w", err)
	}
	defer file.Close()
	reader := tar.NewReader(file)
	var images []string
	entries := 0
	var total int64
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read cbt: %w", err)
		}
		if err := archiveBudgetAdd(&entries, &total, header.Size); err != nil {
			return nil, fmt.Errorf("validate cbt: %w", err)
		}
		if header.FileInfo().IsDir() || !isComicImage(header.Name) {
			continue
		}
		images = append(images, filepath.ToSlash(header.Name))
	}
	sortComicPaths(images)
	return images, nil
}

func getTarAsset(filePath string, assetPath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbt: %w", err)
	}
	defer file.Close()
	reader := tar.NewReader(file)
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	entries := 0
	var total int64
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read cbt: %w", err)
		}
		if err := archiveBudgetAdd(&entries, &total, header.Size); err != nil {
			return nil, fmt.Errorf("validate cbt: %w", err)
		}
		if filepath.ToSlash(header.Name) != assetPath {
			continue
		}
		return bookparser.ReadAllLimit(reader, constants.MaxArchiveAssetSize)
	}
	return nil, fmt.Errorf("cbt asset %q not found", assetPath)
}

func listSevenZipImages(filePath string) ([]string, error) {
	reader, err := sevenzip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cb7: %w", err)
	}
	defer reader.Close()
	if err := sevenZipArchiveBudget(reader.File); err != nil {
		return nil, fmt.Errorf("validate cb7: %w", err)
	}
	var images []string
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !isComicImage(file.Name) {
			continue
		}
		images = append(images, filepath.ToSlash(file.Name))
	}
	sortComicPaths(images)
	return images, nil
}

func getSevenZipAsset(filePath string, assetPath string) ([]byte, error) {
	reader, err := sevenzip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cb7: %w", err)
	}
	defer reader.Close()
	if err := sevenZipArchiveBudget(reader.File); err != nil {
		return nil, fmt.Errorf("validate cb7: %w", err)
	}
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	for _, file := range reader.File {
		if filepath.ToSlash(file.Name) != assetPath {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open cb7 asset: %w", err)
		}
		defer rc.Close()
		return bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
	}
	return nil, fmt.Errorf("cb7 asset %q not found", assetPath)
}

func isComicImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".avif":
		return true
	default:
		return false
	}
}

func sortComicPaths(paths []string) {
	sort.Slice(paths, func(i, j int) bool {
		return naturalLess(paths[i], paths[j])
	})
}

func naturalLess(a, b string) bool {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	ia, ib := 0, 0
	for ia < len(a) && ib < len(b) {
		ra, rb := a[ia], b[ib]
		if isDigit(ra) && isDigit(rb) {
			na, ja := readNumber(a, ia)
			nb, jb := readNumber(b, ib)
			if na != nb {
				return na < nb
			}
			ia, ib = ja, jb
			continue
		}
		if ra != rb {
			return ra < rb
		}
		ia++
		ib++
	}
	return len(a) < len(b)
}

func readNumber(value string, index int) (int, int) {
	number := 0
	for index < len(value) && isDigit(value[index]) {
		number = number*10 + int(value[index]-'0')
		index++
	}
	return number, index
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
