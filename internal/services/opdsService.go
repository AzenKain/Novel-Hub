package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/opds"
)

type OPDSService interface {
	GetRootCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error)
	GetRecentBooks(ctx context.Context, serverURL string, limit int64, claims *response.JWTClaims) (*opds.Feed, error)
	GetOpenSearchDescription(serverURL string) *opds.OpenSearchDescription
	SearchBooksOPDS(ctx context.Context, serverURL string, query string, limit int64, claims *response.JWTClaims) (*opds.Feed, error)
	GetAuthorsCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error)
	GetAuthorBooks(ctx context.Context, serverURL string, authorName string, limit int64, claims *response.JWTClaims) (*opds.Feed, error)
	GetSeriesCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error)
	GetSeriesBooks(ctx context.Context, serverURL string, seriesName string, limit int64, claims *response.JWTClaims) (*opds.Feed, error)
	GetTagsCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error)
	GetTagBooks(ctx context.Context, serverURL string, tagName string, limit int64, claims *response.JWTClaims) (*opds.Feed, error)
	GetOPDS2Catalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (map[string]any, error)
}

type opdsService struct {
	books       BookService
	permissions PermissionCache
}

func NewOPDSService(books BookService, permissions PermissionCache) OPDSService {
	return &opdsService{books: books, permissions: permissions}
}

func parseBookMetadataMap(metadataJSON *string) map[string]any {
	metaMap := map[string]any{}
	if metadataJSON != nil && strings.TrimSpace(*metadataJSON) != "" {
		_ = jsonx.UnmarshalString(*metadataJSON, &metaMap)
	}
	return metaMap
}

func getBookSeries(b *models.BookEntity) string {
	metaMap := parseBookMetadataMap(b.MetadataJSON)
	if series, ok := metaMap["series"].(string); ok && series != "" {
		return series
	}
	return ""
}

func getBookTags(b *models.BookEntity) []string {
	metaMap := parseBookMetadataMap(b.MetadataJSON)
	var tags []string
	if subj, ok := metaMap["subject"]; ok {
		switch v := subj.(type) {
		case string:
			for _, part := range strings.Split(v, ",") {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
		case []any:
			for _, item := range v {
				if str, isStr := item.(string); isStr && strings.TrimSpace(str) != "" {
					tags = append(tags, strings.TrimSpace(str))
				}
			}
		}
	}
	return tags
}

func (s *opdsService) GetRootCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error) {
	now := time.Now()
	return &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:root", Title: "NovelHub OPDS Catalog", Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "search", Href: serverURL + "/api/opds/v1/opensearch.xml", Type: "application/opensearchdescription+xml", Title: "Search NovelHub"},
		},
		Entries: []opds.Entry{
			{ID: "novelhub:recent", Title: "Recent Additions", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + "/api/opds/v1/recent", Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"}}, Content: "Recently added books"},
			{ID: "novelhub:authors", Title: "Authors", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + "/api/opds/v1/authors", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"}}, Content: "Browse books by author"},
			{ID: "novelhub:series", Title: "Series", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + "/api/opds/v1/series", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"}}, Content: "Browse books by series"},
			{ID: "novelhub:tags", Title: "Tags & Genres", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + "/api/opds/v1/tags", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"}}, Content: "Browse books by tags"},
		},
	}, nil
}

func (s *opdsService) GetOpenSearchDescription(serverURL string) *opds.OpenSearchDescription {
	return &opds.OpenSearchDescription{
		Xmlns:          "http://a9.com/-/spec/opensearch/1.1/",
		ShortName:      "NovelHub",
		Description:    "Search books in NovelHub catalog",
		InputEncoding:  "UTF-8",
		OutputEncoding: "UTF-8",
		URL: opds.OpenSearchURL{
			Type:     "application/atom+xml;profile=opds-catalog;kind=acquisition",
			Template: serverURL + "/api/opds/v1/search?q={searchTerms}",
		},
	}
}

func (s *opdsService) visibleBooks(ctx context.Context, limit int64, claims *response.JWTClaims) ([]*models.BookEntity, error) {
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "", "", "", nil, limit)
	if err != nil {
		return nil, err
	}
	readable, allowed := s.books.FilterReadableBooks(ctx, books, claims)
	if !allowed {
		return []*models.BookEntity{}, nil
	}
	visible := readable[:0]
	for _, book := range readable {
		if s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermOPDSRead, map[string]any{"library_id": book.LibraryID}) {
			visible = append(visible, book)
		}
	}
	return visible, nil
}

func (s *opdsService) canAcquire(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	claims = resolveClaims(claims)
	return s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermOPDSDownload, map[string]any{"library_id": book.LibraryID}) && s.books.CanDownloadBook(ctx, book, claims)
}

func (s *opdsService) GetRecentBooks(ctx context.Context, serverURL string, limit int64, claims *response.JWTClaims) (*opds.Feed, error) {
	books, err := s.visibleBooks(ctx, limit, claims)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:recent", Title: "Recent Additions", Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1/recent", Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "search", Href: serverURL + "/api/opds/v1/opensearch.xml", Type: "application/opensearchdescription+xml", Title: "Search NovelHub"},
		},
	}
	for _, book := range books {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, claims))
	}
	return feed, nil
}

func (s *opdsService) SearchBooksOPDS(ctx context.Context, serverURL string, query string, limit int64, claims *response.JWTClaims) (*opds.Feed, error) {
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, query, "", "", "", "", nil, limit)
	if err != nil {
		return nil, err
	}
	readable, allowed := s.books.FilterReadableBooks(ctx, books, claims)
	if !allowed {
		readable = []*models.BookEntity{}
	}
	visible := readable[:0]
	for _, book := range readable {
		if s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermOPDSRead, map[string]any{"library_id": book.LibraryID}) {
			visible = append(visible, book)
		}
	}

	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:search:" + query, Title: "Search Results for " + query, Updated: now,
		TotalResults: len(visible),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1/search?q=" + query, Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, claims))
	}
	return feed, nil
}

func (s *opdsService) GetAuthorsCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error) {
	books, err := s.visibleBooks(ctx, 200, claims)
	if err != nil {
		return nil, err
	}
	authorsMap := make(map[string]int)
	for _, b := range books {
		if b.AuthorName != nil && *b.AuthorName != "" {
			authorsMap[*b.AuthorName]++
		}
	}
	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:authors", Title: "Authors", Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1/authors", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
	}
	for author, count := range authorsMap {
		feed.Entries = append(feed.Entries, opds.Entry{
			ID:      "novelhub:author:" + author,
			Title:   author,
			Updated: now,
			Content: fmt.Sprintf("%d books", count),
			Links: []opds.Link{
				{Rel: "subsection", Href: serverURL + "/api/opds/v1/authors/" + author, Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			},
		})
	}
	return feed, nil
}

func (s *opdsService) GetAuthorBooks(ctx context.Context, serverURL string, authorName string, limit int64, claims *response.JWTClaims) (*opds.Feed, error) {
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", authorName, "", "", "", nil, limit)
	if err != nil {
		return nil, err
	}
	readable, allowed := s.books.FilterReadableBooks(ctx, books, claims)
	if !allowed {
		readable = []*models.BookEntity{}
	}
	visible := readable[:0]
	for _, book := range readable {
		if s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermOPDSRead, map[string]any{"library_id": book.LibraryID}) {
			visible = append(visible, book)
		}
	}
	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:author:" + authorName, Title: "Books by " + authorName, Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1/authors/" + authorName, Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, claims))
	}
	return feed, nil
}

func (s *opdsService) GetSeriesCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error) {
	books, err := s.visibleBooks(ctx, 200, claims)
	if err != nil {
		return nil, err
	}
	seriesMap := make(map[string]int)
	for _, b := range books {
		series := getBookSeries(b)
		if series != "" {
			seriesMap[series]++
		}
	}
	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:series", Title: "Series", Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1/series", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
	}
	for series, count := range seriesMap {
		feed.Entries = append(feed.Entries, opds.Entry{
			ID:      "novelhub:series:" + series,
			Title:   series,
			Updated: now,
			Content: fmt.Sprintf("%d books", count),
			Links: []opds.Link{
				{Rel: "subsection", Href: serverURL + "/api/opds/v1/series/" + series, Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			},
		})
	}
	return feed, nil
}

func (s *opdsService) GetSeriesBooks(ctx context.Context, serverURL string, seriesName string, limit int64, claims *response.JWTClaims) (*opds.Feed, error) {
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", seriesName, "", "", nil, limit)
	if err != nil {
		return nil, err
	}
	readable, allowed := s.books.FilterReadableBooks(ctx, books, claims)
	if !allowed {
		readable = []*models.BookEntity{}
	}
	visible := readable[:0]
	for _, book := range readable {
		if s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermOPDSRead, map[string]any{"library_id": book.LibraryID}) {
			visible = append(visible, book)
		}
	}
	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:series:" + seriesName, Title: "Series: " + seriesName, Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1/series/" + seriesName, Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, claims))
	}
	return feed, nil
}

func (s *opdsService) GetTagsCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error) {
	books, err := s.visibleBooks(ctx, 200, claims)
	if err != nil {
		return nil, err
	}
	tagsMap := make(map[string]int)
	for _, b := range books {
		for _, tag := range getBookTags(b) {
			if tag != "" {
				tagsMap[tag]++
			}
		}
	}
	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:tags", Title: "Tags & Genres", Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1/tags", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
	}
	for tag, count := range tagsMap {
		feed.Entries = append(feed.Entries, opds.Entry{
			ID:      "novelhub:tag:" + tag,
			Title:   tag,
			Updated: now,
			Content: fmt.Sprintf("%d books", count),
			Links: []opds.Link{
				{Rel: "subsection", Href: serverURL + "/api/opds/v1/tags/" + tag, Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			},
		})
	}
	return feed, nil
}

func (s *opdsService) GetTagBooks(ctx context.Context, serverURL string, tagName string, limit int64, claims *response.JWTClaims) (*opds.Feed, error) {
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "", tagName, "", nil, limit)
	if err != nil {
		return nil, err
	}
	readable, allowed := s.books.FilterReadableBooks(ctx, books, claims)
	if !allowed {
		readable = []*models.BookEntity{}
	}
	visible := readable[:0]
	for _, book := range readable {
		if s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermOPDSRead, map[string]any{"library_id": book.LibraryID}) {
			visible = append(visible, book)
		}
	}
	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:tag:" + tagName, Title: "Tag: " + tagName, Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + "/api/opds/v1/tags/" + tagName, Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			{Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, claims))
	}
	return feed, nil
}

func getMimeTypeForFormat(format string) string {
	switch strings.ToLower(strings.TrimPrefix(format, ".")) {
	case "epub":
		return "application/epub+zip"
	case "pdf":
		return "application/pdf"
	case "mobi":
		return "application/x-mobipocket-ebook"
	case "fb2":
		return "application/x-fb2+zip"
	case "cbz":
		return "application/x-cbz"
	case "cbr":
		return "application/x-cbr"
	case "mp3":
		return "audio/mpeg"
	case "m4b":
		return "audio/x-m4b"
	case "flac":
		return "audio/flac"
	default:
		return "application/octet-stream"
	}
}

func (s *opdsService) bookToEntry(ctx context.Context, book *models.BookEntity, serverURL string, claims *response.JWTClaims) opds.Entry {
	summary := ""
	if book.Description != nil {
		summary = *book.Description
	}
	entry := opds.Entry{
		ID:      "novelhub:book:" + book.ID,
		Title:   book.Title,
		Updated: book.UpdatedAt,
		Summary: summary,
		Links: []opds.Link{
			{Rel: "http://opds-spec.org/image/thumbnail", Href: fmt.Sprintf("%s/api/v1/books/%s/cover", serverURL, book.ID), Type: "image/jpeg"},
			{Rel: "http://opds-spec.org/image", Href: fmt.Sprintf("%s/api/v1/books/%s/cover", serverURL, book.ID), Type: "image/jpeg"},
		},
	}
	if s.canAcquire(ctx, book, claims) {
		if len(book.Files) > 0 {
			for _, file := range book.Files {
				mime := getMimeTypeForFormat(file.Format)
				entry.Links = append(entry.Links, opds.Link{
					Rel:   "http://opds-spec.org/acquisition",
					Href:  fmt.Sprintf("%s/api/v1/books/%s/download?file_id=%s", serverURL, book.ID, file.ID),
					Type:  mime,
					Title: strings.ToUpper(file.Format),
				})
			}
		} else {
			entry.Links = append(entry.Links, opds.Link{
				Rel:  "http://opds-spec.org/acquisition",
				Href: fmt.Sprintf("%s/api/v1/books/%s/download", serverURL, book.ID),
				Type: "application/epub+zip",
			})
		}
	}
	if book.AuthorName != nil && *book.AuthorName != "" {
		entry.Author = &opds.Author{Name: *book.AuthorName}
	}
	return entry
}

func (s *opdsService) GetOPDS2Catalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (map[string]any, error) {
	books, err := s.visibleBooks(ctx, 50, claims)
	if err != nil {
		return nil, err
	}
	publications := make([]map[string]any, 0, len(books))
	for _, book := range books {
		links := []map[string]any{{"rel": "http://opds-spec.org/image", "href": fmt.Sprintf("%s/api/v1/books/%s/cover", serverURL, book.ID), "type": "image/jpeg"}}
		if s.canAcquire(ctx, book, claims) {
			if len(book.Files) > 0 {
				for _, file := range book.Files {
					mime := getMimeTypeForFormat(file.Format)
					links = append(links, map[string]any{
						"rel":   "http://opds-spec.org/acquisition",
						"href":  fmt.Sprintf("%s/api/v1/books/%s/download?file_id=%s", serverURL, book.ID, file.ID),
						"type":  mime,
						"title": strings.ToUpper(file.Format),
					})
				}
			} else {
				links = append(links, map[string]any{"rel": "http://opds-spec.org/acquisition", "href": fmt.Sprintf("%s/api/v1/books/%s/download", serverURL, book.ID), "type": "application/epub+zip"})
			}
		}
		metadata := map[string]any{"title": book.Title, "identifier": "urn:novelhub:book:" + book.ID}
		if book.AuthorName != nil && *book.AuthorName != "" {
			metadata["author"] = *book.AuthorName
		}
		publications = append(publications, map[string]any{"metadata": metadata, "links": links})
	}

	searchURL := serverURL + "/api/opds/v1/search?q={searchTerms}"
	return map[string]any{
		"metadata": map[string]any{"title": "NovelHub OPDS 2.0 Catalog"},
		"links": []map[string]any{
			{"rel": "self", "href": serverURL + "/api/opds/v2/catalog", "type": "application/opds+json"},
			{"rel": "search", "href": searchURL, "type": "application/atom+xml;profile=opds-catalog;kind=acquisition"},
		},
		"publications": publications,
	}, nil
}
