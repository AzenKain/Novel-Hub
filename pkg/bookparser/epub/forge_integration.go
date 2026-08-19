package epub

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
)

var (
	metadataRe         = regexp.MustCompile(`(?is)(<metadata\b[^>]*>)(.*?)(</metadata>)`)
	titleRe            = regexp.MustCompile(`(?is)<dc:title[^>]*>.*?</dc:title>`)
	creatorRe          = regexp.MustCompile(`(?is)<dc:creator[^>]*>.*?</dc:creator>`)
	descRe             = regexp.MustCompile(`(?is)<dc:description[^>]*>.*?</dc:description>`)
	publisherRe        = regexp.MustCompile(`(?is)<dc:publisher[^>]*>.*?</dc:publisher>`)
	languageRe         = regexp.MustCompile(`(?is)<dc:language[^>]*>.*?</dc:language>`)
	subjectRe          = regexp.MustCompile(`(?is)<dc:subject[^>]*>.*?</dc:subject>`)
	calibreSeriesRe    = regexp.MustCompile(`(?is)<meta\s+[^>]*name=["']calibre:series["'][^>]*>`)
	calibreSeriesIdxRe = regexp.MustCompile(`(?is)<meta\s+[^>]*name=["']calibre:series_index["'][^>]*>`)
)

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func replaceXMLBlock(re *regexp.Regexp, xmlContent, newContent string) string {
	return re.ReplaceAllString(xmlContent, "${1}\n"+newContent+"\n${3}")
}

func ensureDCTermsModified(xmlContent string) string {
	if strings.Contains(xmlContent, "dcterms:modified") {
		return xmlContent
	}
	nowStr := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	newTag := fmt.Sprintf(`    <meta property="dcterms:modified">%s</meta>`, nowStr)

	match := metadataRe.FindStringSubmatch(xmlContent)
	if len(match) >= 4 {
		return replaceXMLBlock(metadataRe, xmlContent, match[2]+"\n"+newTag)
	}
	return xmlContent
}

func applyMetadataToOPF(opfContent string, meta *bookparser.BookMetadata) string {
	m := metadataRe.FindStringSubmatch(opfContent)
	if len(m) < 4 {
		return opfContent
	}

	metadataBody := m[2]

	if meta.Title != "" {
		if titleRe.MatchString(metadataBody) {
			metadataBody = titleRe.ReplaceAllString(metadataBody, fmt.Sprintf(`<dc:title>%s</dc:title>`, escapeXML(meta.Title)))
		} else {
			metadataBody += fmt.Sprintf("\n    <dc:title>%s</dc:title>", escapeXML(meta.Title))
		}
	}

	if meta.Author != "" {
		if creatorRe.MatchString(metadataBody) {
			metadataBody = creatorRe.ReplaceAllString(metadataBody, fmt.Sprintf(`<dc:creator>%s</dc:creator>`, escapeXML(meta.Author)))
		} else {
			metadataBody += fmt.Sprintf("\n    <dc:creator>%s</dc:creator>", escapeXML(meta.Author))
		}
	}

	if meta.Description != "" {
		if descRe.MatchString(metadataBody) {
			metadataBody = descRe.ReplaceAllString(metadataBody, fmt.Sprintf(`<dc:description>%s</dc:description>`, escapeXML(meta.Description)))
		} else {
			metadataBody += fmt.Sprintf("\n    <dc:description>%s</dc:description>", escapeXML(meta.Description))
		}
	}

	if meta.Publisher != "" {
		if publisherRe.MatchString(metadataBody) {
			metadataBody = publisherRe.ReplaceAllString(metadataBody, fmt.Sprintf(`<dc:publisher>%s</dc:publisher>`, escapeXML(meta.Publisher)))
		} else {
			metadataBody += fmt.Sprintf("\n    <dc:publisher>%s</dc:publisher>", escapeXML(meta.Publisher))
		}
	}

	if meta.Language != "" {
		if languageRe.MatchString(metadataBody) {
			metadataBody = languageRe.ReplaceAllString(metadataBody, fmt.Sprintf(`<dc:language>%s</dc:language>`, escapeXML(meta.Language)))
		} else {
			metadataBody += fmt.Sprintf("\n    <dc:language>%s</dc:language>", escapeXML(meta.Language))
		}
	}

	if len(meta.Subjects) > 0 {
		metadataBody = subjectRe.ReplaceAllString(metadataBody, "")
		for _, subject := range meta.Subjects {
			if trimmed := strings.TrimSpace(subject); trimmed != "" {
				metadataBody += fmt.Sprintf("\n    <dc:subject>%s</dc:subject>", escapeXML(trimmed))
			}
		}
	}

	if meta.Series != "" {
		if calibreSeriesRe.MatchString(metadataBody) {
			metadataBody = calibreSeriesRe.ReplaceAllString(metadataBody, fmt.Sprintf(`<meta name="calibre:series" content="%s"/>`, escapeXML(meta.Series)))
		} else {
			metadataBody += fmt.Sprintf("\n    <meta name=\"calibre:series\" content=\"%s\"/>", escapeXML(meta.Series))
		}

		if meta.SeriesIndex != "" {
			if calibreSeriesIdxRe.MatchString(metadataBody) {
				metadataBody = calibreSeriesIdxRe.ReplaceAllString(metadataBody, fmt.Sprintf(`<meta name="calibre:series_index" content="%s"/>`, escapeXML(meta.SeriesIndex)))
			} else {
				metadataBody += fmt.Sprintf("\n    <meta name=\"calibre:series_index\" content=\"%s\"/>", escapeXML(meta.SeriesIndex))
			}
		}
	}

	opfContent = replaceXMLBlock(metadataRe, opfContent, metadataBody)
	opfContent = ensureDCTermsModified(opfContent)

	return opfContent
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return err
	}

	tmpPath := filePath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	bufOut := bufio.NewWriterSize(out, 2*1024*1024)
	zw := zip.NewWriter(bufOut)

	var opfErr error

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		if f.Name == opfPath {
			rc, err := f.Open()
			if err != nil {
				opfErr = err
				break
			}
			opfBytes, err := bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
			rc.Close()
			if err != nil {
				opfErr = err
				break
			}

			newOpfContent := applyMetadataToOPF(string(opfBytes), meta)

			header := &zip.FileHeader{Name: f.Name, Method: zip.Deflate}
			header.SetMode(f.Mode())
			writer, err := zw.CreateHeader(header)
			if err != nil {
				opfErr = err
				break
			}
			if _, err := writer.Write([]byte(newOpfContent)); err != nil {
				opfErr = err
				break
			}
		} else {
			rc, err := f.Open()
			if err != nil {
				opfErr = err
				break
			}

			header := &zip.FileHeader{Name: f.Name, Method: f.Method}
			header.SetMode(f.Mode())
			writer, err := zw.CreateHeader(header)
			if err != nil {
				rc.Close()
				opfErr = err
				break
			}
			_, err = io.Copy(writer, rc)
			rc.Close()
			if err != nil {
				opfErr = err
				break
			}
		}
	}

	if err := zw.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := bufOut.Flush(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if opfErr != nil {
		os.Remove(tmpPath)
		return opfErr
	}

	out.Close()
	r.Close()

	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}
