package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/opds"
)

type OPDSService interface {
	GetRootCatalog(ctx context.Context, serverURL, basePath string, claims *response.JWTClaims) (*opds.Feed, error)
	GetAllBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetRecentBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetHotBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetRandomBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetOpenSearchDescription(serverURL, basePath string) *opds.OpenSearchDescription
	SearchBooksOPDS(ctx context.Context, serverURL, basePath string, query string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetAuthorsCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetAuthorBooks(ctx context.Context, serverURL, basePath string, authorName string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetSeriesCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetSeriesBooks(ctx context.Context, serverURL, basePath string, seriesName string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetTagsCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)
	GetTagBooks(ctx context.Context, serverURL, basePath string, tagName string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error)

	GetOPDS2Catalog(ctx context.Context, serverURL, basePath string, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2AllBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2RecentBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2HotBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2RandomBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2AuthorsCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2AuthorBooks(ctx context.Context, serverURL, basePath string, authorName string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2SeriesCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2SeriesBooks(ctx context.Context, serverURL, basePath string, seriesName string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2TagsCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2TagBooks(ctx context.Context, serverURL, basePath string, tagName string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)
	GetOPDS2Search(ctx context.Context, serverURL, basePath string, query string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error)

	GetBookCoverPath(ctx context.Context, bookID string, claims *response.JWTClaims) (string, error)
	GetBookFileForDownload(ctx context.Context, bookID, fileID string, claims *response.JWTClaims) (string, string, error)
}

type opdsService struct {
	books           BookService
	metadataService MetadataService
	settings        SettingsService
	permissions     PermissionCache
}

func NewOPDSService(books BookService, metadataService MetadataService, settings SettingsService, permissions PermissionCache) OPDSService {
	return &opdsService{
		books:           books,
		metadataService: metadataService,
		settings:        settings,
		permissions:     permissions,
	}
}

func resolveBasePath(basePath string) string {
	clean := strings.TrimRight(strings.TrimSpace(basePath), "/")
	if clean == "" {
		return "/api/opds"
	}
	return clean
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

func (s *opdsService) GetRootCatalog(ctx context.Context, serverURL, basePath string, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	now := time.Now()
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:root", Title: "NovelHub OPDS Catalog", Updated: now,
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
			{Rel: "search", Href: serverURL + basePath + "/v1/opensearch.xml", Type: "application/opensearchdescription+xml", Title: "Search NovelHub"},
		},
		Entries: []opds.Entry{
			{ID: "novelhub:books", Title: "All Books", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + basePath + "/v1/books", Type: opds.MimeTypeAcquisition}}, Content: "Browse all books"},
			{ID: "novelhub:recent", Title: "Recent Additions", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + basePath + "/v1/recent", Type: opds.MimeTypeAcquisition}}, Content: "Recently added books"},
		},
	}

	showHot := false
	showRandom := false
	if s.settings != nil {
		if pub, err := s.settings.Public(ctx); err == nil && pub != nil {
			showHot = pub.HomeSections.TopBooks
			showRandom = pub.HomeSections.RandomBooks
		}
	}

	if showHot {
		feed.Entries = append(feed.Entries, opds.Entry{
			ID: "novelhub:hot", Title: "Hot Books", Updated: now,
			Links:   []opds.Link{{Rel: "subsection", Href: serverURL + basePath + "/v1/hot", Type: opds.MimeTypeAcquisition}},
			Content: "Popular and trending books",
		})
	}
	if showRandom {
		feed.Entries = append(feed.Entries, opds.Entry{
			ID: "novelhub:random", Title: "Random Books", Updated: now,
			Links:   []opds.Link{{Rel: "subsection", Href: serverURL + basePath + "/v1/random", Type: opds.MimeTypeAcquisition}},
			Content: "Random selection of books",
		})
	}

	feed.Entries = append(feed.Entries,
		opds.Entry{ID: "novelhub:authors", Title: "Authors", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + basePath + "/v1/authors", Type: opds.MimeTypeAtom}}, Content: "Browse books by author"},
		opds.Entry{ID: "novelhub:series", Title: "Series", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + basePath + "/v1/series", Type: opds.MimeTypeAtom}}, Content: "Browse books by series"},
		opds.Entry{ID: "novelhub:tags", Title: "Tags & Genres", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + basePath + "/v1/tags", Type: opds.MimeTypeAtom}}, Content: "Browse books by tags"},
	)

	return feed, nil
}

func (s *opdsService) GetOpenSearchDescription(serverURL, basePath string) *opds.OpenSearchDescription {
	basePath = resolveBasePath(basePath)
	return &opds.OpenSearchDescription{
		Xmlns:          "http://a9.com/-/spec/opensearch/1.1/",
		ShortName:      "NovelHub",
		Description:    "Search books in NovelHub catalog",
		InputEncoding:  "UTF-8",
		OutputEncoding: "UTF-8",
		URL: opds.OpenSearchURL{
			Type:     opds.MimeTypeAcquisition,
			Template: serverURL + basePath + "/v1/search?q={searchTerms}",
		},
	}
}

type visiblePage struct {
	Books      []*models.BookEntity
	NextCursor string
}

func (s *opdsService) filterVisible(ctx context.Context, books []*models.BookEntity, claims *response.JWTClaims) []*models.BookEntity {
	readable, allowed := s.books.FilterReadableBooks(ctx, books, claims)
	if !allowed {
		return []*models.BookEntity{}
	}
	visible := make([]*models.BookEntity, 0, len(readable))
	for _, book := range readable {
		if s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermOPDSRead, map[string]any{"library_id": book.LibraryID}) {
			visible = append(visible, book)
		}
	}
	return visible
}

func nextCursor(rows []*models.BookEntity, limit int64) string {
	if int64(len(rows)) < limit || len(rows) == 0 {
		return ""
	}
	last := rows[len(rows)-1]
	return convert.EncodeCursor(last.CreatedAt, last.ID)
}

func decodeBookCursor(cursor string) (*time.Time, string) {
	if cursor == "" {
		return nil, ""
	}
	parts := convert.DecodeCursor(cursor)
	if len(parts) != 2 {
		return nil, ""
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, ""
	}
	return &t, parts[1]
}

func (s *opdsService) visibleBooks(ctx context.Context, limit int64, claims *response.JWTClaims) ([]*models.BookEntity, error) {
	page, err := s.visibleBooksPage(ctx, request.OPDSPageDto{Limit: limit}, claims)
	if err != nil {
		return nil, err
	}
	return page.Books, nil
}

func (s *opdsService) visibleBooksPage(ctx context.Context, q request.OPDSPageDto, claims *response.JWTClaims) (visiblePage, error) {
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "ExcludeAudiobooks", "", "", "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return visiblePage{}, err
	}
	return visiblePage{Books: s.filterVisible(ctx, books, claims), NextCursor: nextCursor(books, q.Limit)}, nil
}

func (s *opdsService) canAcquire(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	claims = resolveClaims(claims)
	return s.books.CanReadBook(ctx, book, claims)
}

func appendNextLink(feed *opds.Feed, serverURL, selfPath, cursor string) {
	if cursor == "" {
		return
	}
	separator := "?"
	if strings.Contains(selfPath, "?") {
		separator = "&"
	}
	feed.Links = append(feed.Links, opds.Link{
		Rel:  "next",
		Href: serverURL + selfPath + separator + "cursor=" + url.QueryEscape(cursor),
		Type: opds.MimeTypeAcquisition,
	})
}

func (s *opdsService) GetAllBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	page, err := s.visibleBooksPage(ctx, q, claims)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	selfPath := basePath + "/v1/books"
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:books", Title: "All Books", Updated: now,
		ItemsPerPage: int(q.Limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAcquisition},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
			{Rel: "search", Href: serverURL + basePath + "/v1/opensearch.xml", Type: "application/opensearchdescription+xml", Title: "Search NovelHub"},
		},
	}
	for _, book := range page.Books {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, basePath, claims))
	}
	appendNextLink(feed, serverURL, selfPath, page.NextCursor)
	return feed, nil
}

func (s *opdsService) GetRecentBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	page, err := s.visibleBooksPage(ctx, q, claims)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	selfPath := basePath + "/v1/recent"
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:recent", Title: "Recent Additions", Updated: now,
		ItemsPerPage: int(q.Limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAcquisition},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
			{Rel: "search", Href: serverURL + basePath + "/v1/opensearch.xml", Type: "application/opensearchdescription+xml", Title: "Search NovelHub"},
		},
	}
	for _, book := range page.Books {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, basePath, claims))
	}
	appendNextLink(feed, serverURL, selfPath, page.NextCursor)
	return feed, nil
}

func (s *opdsService) GetHotBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "hot", "", "ExcludeAudiobooks", "", "", "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	now := time.Now()
	selfPath := basePath + "/v1/hot"
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:hot", Title: "Hot Books", Updated: now,
		ItemsPerPage: int(q.Limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAcquisition},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
			{Rel: "search", Href: serverURL + basePath + "/v1/opensearch.xml", Type: "application/opensearchdescription+xml", Title: "Search NovelHub"},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, basePath, claims))
	}
	appendNextLink(feed, serverURL, selfPath, nextCursor(books, q.Limit))
	return feed, nil
}

func (s *opdsService) GetRandomBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	limit := q.Limit
	if limit <= 0 {
		limit = int64(constants.OPDSDefaultPageSize)
	}
	books, err := s.books.SearchBooks(ctx, nil, nil, "random", "", "ExcludeAudiobooks", "", "", "", "", limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	now := time.Now()
	selfPath := basePath + "/v1/random"
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:random", Title: "Random Books", Updated: now,
		ItemsPerPage: int(limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAcquisition},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
			{Rel: "search", Href: serverURL + basePath + "/v1/opensearch.xml", Type: "application/opensearchdescription+xml", Title: "Search NovelHub"},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, basePath, claims))
	}
	return feed, nil
}

func (s *opdsService) SearchBooksOPDS(ctx context.Context, serverURL, basePath string, query string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, &query, "", "", "ExcludeAudiobooks", "", "", "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)

	now := time.Now()
	selfPath := basePath + "/v1/search?q=" + url.QueryEscape(query)
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:search:" + query, Title: "Search Results for " + query, Updated: now,
		ItemsPerPage: int(q.Limit),
		TotalResults: len(visible),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAcquisition},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, basePath, claims))
	}
	appendNextLink(feed, serverURL, selfPath, nextCursor(books, q.Limit))
	return feed, nil
}

func (s *opdsService) GetAuthorsCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	now := time.Now()
	limit := q.Limit
	if limit <= 0 {
		limit = int64(constants.OPDSDefaultPageSize)
	}
	selfPath := basePath + "/v1/authors"
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:authors", Title: "Authors", Updated: now,
		ItemsPerPage: int(limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAtom},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
		},
	}

	if s.metadataService != nil {
		res, err := s.metadataService.ListAuthors(ctx, &request.MetadataFacetDto{Limit: int(limit), Cursor: q.Cursor}, claims)
		if err != nil {
			return nil, err
		}
		if data, ok := res.Data.([]*response.MetadataCountResponse); ok {
			for _, item := range data {
				feed.Entries = append(feed.Entries, opds.Entry{
					ID:      "novelhub:author:" + item.ID,
					Title:   item.Name,
					Updated: now,
					Content: fmt.Sprintf("%d books", item.BookCount),
					Links: []opds.Link{
						{Rel: "subsection", Href: serverURL + basePath + "/v1/authors/" + url.PathEscape(item.Name), Type: opds.MimeTypeAcquisition},
					},
				})
			}
		}
		if res.Pagination != nil && res.Pagination.NextCursor != "" {
			feed.Links = append(feed.Links, opds.Link{
				Rel:  "next",
				Href: serverURL + selfPath + "?cursor=" + url.QueryEscape(res.Pagination.NextCursor),
				Type: opds.MimeTypeAtom,
			})
		}
		return feed, nil
	}

	page, err := s.visibleBooksPage(ctx, q, claims)
	if err != nil {
		return nil, err
	}
	authorsMap := make(map[string]int)
	for _, b := range page.Books {
		if b.AuthorName != nil && *b.AuthorName != "" {
			authorsMap[*b.AuthorName]++
		}
	}
	for author, count := range authorsMap {
		feed.Entries = append(feed.Entries, opds.Entry{
			ID:      "novelhub:author:" + author,
			Title:   author,
			Updated: now,
			Content: fmt.Sprintf("%d books", count),
			Links: []opds.Link{
				{Rel: "subsection", Href: serverURL + basePath + "/v1/authors/" + url.PathEscape(author), Type: opds.MimeTypeAcquisition},
			},
		})
	}
	appendNextLink(feed, serverURL, selfPath, page.NextCursor)
	return feed, nil
}

func (s *opdsService) GetAuthorBooks(ctx context.Context, serverURL, basePath string, authorName string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "ExcludeAudiobooks", "author", authorName, "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	now := time.Now()
	selfPath := basePath + "/v1/authors/" + url.PathEscape(authorName)
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:author:" + authorName, Title: "Books by " + authorName, Updated: now,
		ItemsPerPage: int(q.Limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAcquisition},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, basePath, claims))
	}
	appendNextLink(feed, serverURL, selfPath, nextCursor(books, q.Limit))
	return feed, nil
}

func (s *opdsService) GetSeriesCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	now := time.Now()
	limit := q.Limit
	if limit <= 0 {
		limit = int64(constants.OPDSDefaultPageSize)
	}
	selfPath := basePath + "/v1/series"
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:series", Title: "Series", Updated: now,
		ItemsPerPage: int(limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAtom},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
		},
	}

	if s.metadataService != nil {
		res, err := s.metadataService.ListSeries(ctx, &request.MetadataFacetDto{Limit: int(limit), Cursor: q.Cursor}, claims)
		if err != nil {
			return nil, err
		}
		if data, ok := res.Data.([]*response.MetadataCountResponse); ok {
			for _, item := range data {
				feed.Entries = append(feed.Entries, opds.Entry{
					ID:      "novelhub:series:" + item.ID,
					Title:   item.Name,
					Updated: now,
					Content: fmt.Sprintf("%d books", item.BookCount),
					Links: []opds.Link{
						{Rel: "subsection", Href: serverURL + basePath + "/v1/series/" + url.PathEscape(item.Name), Type: opds.MimeTypeAcquisition},
					},
				})
			}
		}
		if res.Pagination != nil && res.Pagination.NextCursor != "" {
			feed.Links = append(feed.Links, opds.Link{
				Rel:  "next",
				Href: serverURL + selfPath + "?cursor=" + url.QueryEscape(res.Pagination.NextCursor),
				Type: opds.MimeTypeAtom,
			})
		}
		return feed, nil
	}

	page, err := s.visibleBooksPage(ctx, q, claims)
	if err != nil {
		return nil, err
	}
	seriesMap := make(map[string]int)
	for _, b := range page.Books {
		series := getBookSeries(b)
		if series != "" {
			seriesMap[series]++
		}
	}
	for series, count := range seriesMap {
		feed.Entries = append(feed.Entries, opds.Entry{
			ID:      "novelhub:series:" + series,
			Title:   series,
			Updated: now,
			Content: fmt.Sprintf("%d books", count),
			Links: []opds.Link{
				{Rel: "subsection", Href: serverURL + basePath + "/v1/series/" + url.PathEscape(series), Type: opds.MimeTypeAcquisition},
			},
		})
	}
	appendNextLink(feed, serverURL, selfPath, page.NextCursor)
	return feed, nil
}

func (s *opdsService) GetSeriesBooks(ctx context.Context, serverURL, basePath string, seriesName string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "ExcludeAudiobooks", "series", seriesName, "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	now := time.Now()
	selfPath := basePath + "/v1/series/" + url.PathEscape(seriesName)
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:series:" + seriesName, Title: "Series: " + seriesName, Updated: now,
		ItemsPerPage: int(q.Limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAcquisition},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, basePath, claims))
	}
	appendNextLink(feed, serverURL, selfPath, nextCursor(books, q.Limit))
	return feed, nil
}

func (s *opdsService) GetTagsCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	now := time.Now()
	limit := q.Limit
	if limit <= 0 {
		limit = int64(constants.OPDSDefaultPageSize)
	}
	selfPath := basePath + "/v1/tags"
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:tags", Title: "Tags & Genres", Updated: now,
		ItemsPerPage: int(limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAtom},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
		},
	}

	if s.metadataService != nil {
		res, err := s.metadataService.ListTags(ctx, &request.MetadataFacetDto{Limit: int(limit), Cursor: q.Cursor}, claims)
		if err != nil {
			return nil, err
		}
		if data, ok := res.Data.([]*response.MetadataCountResponse); ok {
			for _, item := range data {
				feed.Entries = append(feed.Entries, opds.Entry{
					ID:      "novelhub:tag:" + item.ID,
					Title:   item.Name,
					Updated: now,
					Content: fmt.Sprintf("%d books", item.BookCount),
					Links: []opds.Link{
						{Rel: "subsection", Href: serverURL + basePath + "/v1/tags/" + url.PathEscape(item.Name), Type: opds.MimeTypeAcquisition},
					},
				})
			}
		}
		if res.Pagination != nil && res.Pagination.NextCursor != "" {
			feed.Links = append(feed.Links, opds.Link{
				Rel:  "next",
				Href: serverURL + selfPath + "?cursor=" + url.QueryEscape(res.Pagination.NextCursor),
				Type: opds.MimeTypeAtom,
			})
		}
		return feed, nil
	}

	page, err := s.visibleBooksPage(ctx, q, claims)
	if err != nil {
		return nil, err
	}
	tagsMap := make(map[string]int)
	for _, b := range page.Books {
		for _, tag := range getBookTags(b) {
			if tag != "" {
				tagsMap[tag]++
			}
		}
	}
	for tag, count := range tagsMap {
		feed.Entries = append(feed.Entries, opds.Entry{
			ID:      "novelhub:tag:" + tag,
			Title:   tag,
			Updated: now,
			Content: fmt.Sprintf("%d books", count),
			Links: []opds.Link{
				{Rel: "subsection", Href: serverURL + basePath + "/v1/tags/" + url.PathEscape(tag), Type: opds.MimeTypeAcquisition},
			},
		})
	}
	appendNextLink(feed, serverURL, selfPath, page.NextCursor)
	return feed, nil
}

func (s *opdsService) GetTagBooks(ctx context.Context, serverURL, basePath string, tagName string, q request.OPDSPageDto, claims *response.JWTClaims) (*opds.Feed, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "ExcludeAudiobooks", "tag", tagName, "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	now := time.Now()
	selfPath := basePath + "/v1/tags/" + url.PathEscape(tagName)
	feed := &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:tag:" + tagName, Title: "Tag: " + tagName, Updated: now,
		ItemsPerPage: int(q.Limit),
		Links: []opds.Link{
			{Rel: "self", Href: serverURL + selfPath, Type: opds.MimeTypeAcquisition},
			{Rel: "start", Href: serverURL + basePath + "/v1", Type: opds.MimeTypeAtom},
		},
	}
	for _, book := range visible {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, basePath, claims))
	}
	appendNextLink(feed, serverURL, selfPath, nextCursor(books, q.Limit))
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
	case "cbr", "rar":
		return "application/x-cbr"
	case "cb7", "7z":
		return "application/x-cb7"
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

func resolveBookCover(serverURL, basePath, bookID string, rawCoverURL *string, updatedAt *time.Time, defaultVersion string) (string, string) {
	coverExt := ".jpg"
	if rawCoverURL != nil && strings.TrimSpace(*rawCoverURL) != "" {
		coverExt = filepath.Ext(*rawCoverURL)
	}
	coverMime := "image/jpeg"
	switch strings.ToLower(coverExt) {
	case ".png":
		coverMime = "image/png"
	case ".webp":
		coverMime = "image/webp"
	case ".gif":
		coverMime = "image/gif"
	case ".svg":
		coverMime = "image/svg+xml"
	}

	if rawCoverURL != nil && strings.TrimSpace(*rawCoverURL) != "" {
		trimmed := strings.TrimSpace(*rawCoverURL)
		var u string
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			u = trimmed
		} else if strings.HasPrefix(trimmed, "/") {
			u = serverURL + trimmed
		} else {
			u = serverURL + "/" + trimmed
		}
		if updatedAt != nil && !updatedAt.IsZero() {
			sep := "?"
			if strings.Contains(u, "?") {
				sep = "&"
			}
			u = fmt.Sprintf("%s%st=%d", u, sep, updatedAt.UnixMilli())
		}
		return u, coverMime
	}

	return fmt.Sprintf("%s%s/%s/books/%s/cover", serverURL, basePath, defaultVersion, bookID), coverMime
}

func (s *opdsService) bookToEntry(ctx context.Context, book *models.BookEntity, serverURL, basePath string, claims *response.JWTClaims) opds.Entry {
	summary := ""
	if book.Description != nil {
		summary = *book.Description
	}
	entry := opds.Entry{
		ID:      "novelhub:book:" + book.ID,
		Title:   book.Title,
		Updated: book.UpdatedAt,
		Summary: summary,
		Links:   []opds.Link{},
	}

	coverURL, coverMime := resolveBookCover(serverURL, basePath, book.ID, book.CoverURL, &book.UpdatedAt, "v1")
	entry.Links = append(entry.Links,
		opds.Link{Rel: "http://opds-spec.org/image/thumbnail", Href: coverURL, Type: coverMime},
		opds.Link{Rel: "http://opds-spec.org/image", Href: coverURL, Type: coverMime},
	)

	if s.canAcquire(ctx, book, claims) {
		if len(book.Files) > 0 {
			for _, file := range book.Files {
				mime := getMimeTypeForFormat(file.Format)
				entry.Links = append(entry.Links, opds.Link{
					Rel:   "http://opds-spec.org/acquisition",
					Href:  fmt.Sprintf("%s%s/v1/books/%s/download?file_id=%s", serverURL, basePath, book.ID, file.ID),
					Type:  mime,
					Title: strings.ToUpper(file.Format),
				})
			}
		} else {
			entry.Links = append(entry.Links, opds.Link{
				Rel:  "http://opds-spec.org/acquisition",
				Href: fmt.Sprintf("%s%s/v1/books/%s/download", serverURL, basePath, book.ID),
				Type: "application/epub+zip",
			})
		}
	}
	if book.AuthorName != nil && *book.AuthorName != "" {
		entry.Author = &opds.Author{Name: *book.AuthorName}
	}
	return entry
}

func (s *opdsService) GetBookCoverPath(ctx context.Context, bookID string, claims *response.JWTClaims) (string, error) {
	return s.books.GetBookCoverPath(ctx, bookID, claims)
}

func (s *opdsService) GetBookFileForDownload(ctx context.Context, bookID, fileID string, claims *response.JWTClaims) (string, string, error) {
	claims = resolveClaims(claims)
	book, err := s.books.GetBook(ctx, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return "", "", err
	}
	if !s.books.CanReadBook(ctx, book, claims) {
		return "", "", apperrors.New(apperrors.ErrForbidden, "Access denied")
	}

	var file *models.BookFileEntity
	if fileID != "" {
		file, err = s.books.GetBookFile(ctx, bookID, fileID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", "", apperrors.New(apperrors.ErrNotFound, "Book file not found")
			}
			return "", "", err
		}
	} else {
		files, err := s.books.ListBookFiles(ctx, bookID)
		if err != nil || len(files) == 0 {
			return "", "", apperrors.New(apperrors.ErrNotFound, "No files available for this book")
		}
		file = files[0]
	}

	ext := strings.ToLower(filepath.Ext(file.Path))
	if ext == "" {
		ext = "." + strings.ToLower(file.Format)
	}
	downloadName := s.books.SafeDownloadFilename(book.Title, ext)
	return file.Path, downloadName, nil
}
