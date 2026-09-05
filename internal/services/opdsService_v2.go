package services

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/constants"
	"novelhub/pkg/opds"
)

func (s *opdsService) bookToOPDS2Publication(ctx context.Context, book *models.BookEntity, serverURL, basePath string, claims *response.JWTClaims) map[string]any {
	coverURL, coverMime := resolveBookCover(serverURL, basePath, book.ID, book.CoverURL, &book.UpdatedAt, "v2")

	images := make([]map[string]any, 0, 1)
	links := make([]map[string]any, 0, 4)

	images = append(images, map[string]any{
		"href": coverURL,
		"type": coverMime,
	})
	links = append(links, map[string]any{
		"rel":  "http://opds-spec.org/image",
		"href": coverURL,
		"type": coverMime,
	})

	if s.canAcquire(ctx, book, claims) {
		if len(book.Files) > 0 {
			for _, file := range book.Files {
				mime := getMimeTypeForFormat(file.Format)
				links = append(links, map[string]any{
					"rel":   "http://opds-spec.org/acquisition",
					"href":  fmt.Sprintf("%s%s/v2/books/%s/download?file_id=%s", serverURL, basePath, book.ID, file.ID),
					"type":  mime,
					"title": strings.ToUpper(file.Format),
				})
			}
		} else {
			links = append(links, map[string]any{
				"rel":  "http://opds-spec.org/acquisition",
				"href": fmt.Sprintf("%s%s/v2/books/%s/download", serverURL, basePath, book.ID),
				"type": "application/epub+zip",
			})
		}
	}

	metadata := map[string]any{
		"@type":      "http://schema.org/Book",
		"title":      book.Title,
		"identifier": "urn:novelhub:book:" + book.ID,
		"modified":   book.UpdatedAt.Format(time.RFC3339),
	}
	metaMap := parseBookMetadataMap(book.MetadataJSON)
	pubDate := book.CreatedAt.Format("2006-01-02")
	if d, ok := metaMap["date"].(string); ok && strings.TrimSpace(d) != "" {
		pubDate = strings.TrimSpace(d)
	}
	metadata["published"] = pubDate
	if book.AuthorName != nil && *book.AuthorName != "" {
		metadata["author"] = *book.AuthorName
	}
	if book.Description != nil && *book.Description != "" {
		metadata["summary"] = *book.Description
		metadata["description"] = *book.Description
	}
	if series := getBookSeries(book); series != "" {
		metadata["belongsTo"] = map[string]any{"series": series}
	}
	if tags := getBookTags(book); len(tags) > 0 {
		metadata["subject"] = tags
	}

	return map[string]any{
		"metadata": metadata,
		"images":   images,
		"links":    links,
	}
}

func (s *opdsService) GetOPDS2Catalog(ctx context.Context, serverURL, basePath string, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	books, err := s.visibleBooks(ctx, 50, claims)
	if err != nil {
		return nil, err
	}
	publications := make([]map[string]any, 0, len(books))
	for _, book := range books {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	navigation := []map[string]any{
		{"title": "All Books", "href": serverURL + basePath + "/v2/books", "type": opds.MimeTypeOPDS2, "rel": "subsection"},
		{"title": "Recent Additions", "href": serverURL + basePath + "/v2/recent", "type": opds.MimeTypeOPDS2, "rel": "subsection"},
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
		navigation = append(navigation, map[string]any{
			"title": "Hot Books", "href": serverURL + basePath + "/v2/hot", "type": opds.MimeTypeOPDS2, "rel": "subsection",
		})
	}
	if showRandom {
		navigation = append(navigation, map[string]any{
			"title": "Random Books", "href": serverURL + basePath + "/v2/random", "type": opds.MimeTypeOPDS2, "rel": "subsection",
		})
	}

	navigation = append(navigation,
		map[string]any{"title": "Authors", "href": serverURL + basePath + "/v2/authors", "type": opds.MimeTypeOPDS2, "rel": "subsection"},
		map[string]any{"title": "Series", "href": serverURL + basePath + "/v2/series", "type": opds.MimeTypeOPDS2, "rel": "subsection"},
		map[string]any{"title": "Tags & Genres", "href": serverURL + basePath + "/v2/tags", "type": opds.MimeTypeOPDS2, "rel": "subsection"},
	)

	return map[string]any{
		"metadata": map[string]any{
			"title":         "NovelHub OPDS 2.0 Catalog",
			"numberOfItems": len(publications),
		},
		"links": []map[string]any{
			{"rel": "self", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
			{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
			{"rel": "search", "href": serverURL + basePath + "/v2/search?q={searchTerms}", "type": opds.MimeTypeOPDS2, "templated": true},
		},
		"navigation":   navigation,
		"publications": publications,
	}, nil
}

func (s *opdsService) GetOPDS2AllBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	page, err := s.visibleBooksPage(ctx, q, claims)
	if err != nil {
		return nil, err
	}
	publications := make([]map[string]any, 0, len(page.Books))
	for _, book := range page.Books {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	selfPath := basePath + "/v2/books"
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	if page.NextCursor != "" {
		links = append(links, map[string]any{
			"rel":  "next",
			"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(page.NextCursor),
			"type": opds.MimeTypeOPDS2,
		})
	}

	meta := map[string]any{
		"title":        "All Books",
		"itemsPerPage": len(publications),
	}
	if page.NextCursor == "" {
		meta["numberOfItems"] = len(publications)
	}

	return map[string]any{
		"metadata":     meta,
		"links":        links,
		"publications": publications,
	}, nil
}

func (s *opdsService) GetOPDS2RecentBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	page, err := s.visibleBooksPage(ctx, q, claims)
	if err != nil {
		return nil, err
	}
	publications := make([]map[string]any, 0, len(page.Books))
	for _, book := range page.Books {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	selfPath := basePath + "/v2/recent"
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	if page.NextCursor != "" {
		links = append(links, map[string]any{
			"rel":  "next",
			"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(page.NextCursor),
			"type": opds.MimeTypeOPDS2,
		})
	}

	meta := map[string]any{
		"title":        "Recent Additions",
		"itemsPerPage": len(publications),
	}
	if page.NextCursor == "" {
		meta["numberOfItems"] = len(publications)
	}

	return map[string]any{
		"metadata":     meta,
		"links":        links,
		"publications": publications,
	}, nil
}

func (s *opdsService) GetOPDS2HotBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "hot", "", "ExcludeAudiobooks", "", "", "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	publications := make([]map[string]any, 0, len(visible))
	for _, book := range visible {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	selfPath := basePath + "/v2/hot"
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	cursor := nextCursor(books, q.Limit)
	if cursor != "" {
		links = append(links, map[string]any{
			"rel":  "next",
			"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(cursor),
			"type": opds.MimeTypeOPDS2,
		})
	}

	meta := map[string]any{
		"title":        "Hot Books",
		"itemsPerPage": len(publications),
	}
	if cursor == "" {
		meta["numberOfItems"] = len(publications)
	}

	return map[string]any{
		"metadata":     meta,
		"links":        links,
		"publications": publications,
	}, nil
}

func (s *opdsService) GetOPDS2RandomBooks(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
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
	publications := make([]map[string]any, 0, len(visible))
	for _, book := range visible {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	selfPath := basePath + "/v2/random"
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}

	return map[string]any{
		"metadata": map[string]any{
			"title":         "Random Books",
			"itemsPerPage":  len(publications),
			"numberOfItems": len(publications),
		},
		"links":        links,
		"publications": publications,
	}, nil
}

func (s *opdsService) GetOPDS2AuthorsCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	limit := q.Limit
	if limit <= 0 {
		limit = int64(constants.MaxPaginationLimit)
	}
	selfPath := basePath + "/v2/authors"
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	var navigation []map[string]any
	hasMore := false

	if s.metadataService != nil {
		res, err := s.metadataService.ListAuthors(ctx, &request.MetadataFacetDto{Limit: int(limit), Cursor: q.Cursor}, claims)
		if err != nil {
			return nil, err
		}
		if data, ok := res.Data.([]*response.MetadataCountResponse); ok {
			navigation = make([]map[string]any, 0, len(data))
			for _, item := range data {
				navigation = append(navigation, map[string]any{
					"title":         item.Name,
					"href":          serverURL + basePath + "/v2/authors/" + url.PathEscape(item.Name),
					"type":          opds.MimeTypeOPDS2,
					"rel":           "subsection",
					"numberOfItems": item.BookCount,
				})
			}
		}
		if res.Pagination != nil && res.Pagination.NextCursor != "" {
			hasMore = true
			links = append(links, map[string]any{
				"rel":  "next",
				"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(res.Pagination.NextCursor),
				"type": opds.MimeTypeOPDS2,
			})
		}
	} else {
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
		navigation = make([]map[string]any, 0, len(authorsMap))
		for author, count := range authorsMap {
			navigation = append(navigation, map[string]any{
				"title":         author,
				"href":          serverURL + basePath + "/v2/authors/" + url.PathEscape(author),
				"type":          opds.MimeTypeOPDS2,
				"rel":           "subsection",
				"numberOfItems": count,
			})
		}
		if page.NextCursor != "" {
			hasMore = true
			links = append(links, map[string]any{
				"rel":  "next",
				"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(page.NextCursor),
				"type": opds.MimeTypeOPDS2,
			})
		}
	}

	meta := map[string]any{
		"title":        "Authors",
		"itemsPerPage": len(navigation),
	}
	if !hasMore {
		meta["numberOfItems"] = len(navigation)
	}

	return map[string]any{
		"metadata":   meta,
		"links":      links,
		"navigation": navigation,
	}, nil
}

func (s *opdsService) GetOPDS2AuthorBooks(ctx context.Context, serverURL, basePath string, authorName string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "ExcludeAudiobooks", "author", authorName, "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	publications := make([]map[string]any, 0, len(visible))
	for _, book := range visible {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	selfPath := basePath + "/v2/authors/" + url.PathEscape(authorName)
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	cursor := nextCursor(books, q.Limit)
	if cursor != "" {
		links = append(links, map[string]any{
			"rel":  "next",
			"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(cursor),
			"type": opds.MimeTypeOPDS2,
		})
	}

	meta := map[string]any{
		"title":        "Books by " + authorName,
		"itemsPerPage": len(publications),
	}
	if cursor == "" {
		meta["numberOfItems"] = len(publications)
	}

	return map[string]any{
		"metadata":     meta,
		"links":        links,
		"publications": publications,
	}, nil
}

func (s *opdsService) GetOPDS2SeriesCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	limit := q.Limit
	if limit <= 0 {
		limit = int64(constants.MaxPaginationLimit)
	}
	selfPath := basePath + "/v2/series"
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	var navigation []map[string]any
	hasMore := false

	if s.metadataService != nil {
		res, err := s.metadataService.ListSeries(ctx, &request.MetadataFacetDto{Limit: int(limit), Cursor: q.Cursor}, claims)
		if err != nil {
			return nil, err
		}
		if data, ok := res.Data.([]*response.MetadataCountResponse); ok {
			navigation = make([]map[string]any, 0, len(data))
			for _, item := range data {
				navigation = append(navigation, map[string]any{
					"title":         item.Name,
					"href":          serverURL + basePath + "/v2/series/" + url.PathEscape(item.Name),
					"type":          opds.MimeTypeOPDS2,
					"rel":           "subsection",
					"numberOfItems": item.BookCount,
				})
			}
		}
		if res.Pagination != nil && res.Pagination.NextCursor != "" {
			hasMore = true
			links = append(links, map[string]any{
				"rel":  "next",
				"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(res.Pagination.NextCursor),
				"type": opds.MimeTypeOPDS2,
			})
		}
	} else {
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
		navigation = make([]map[string]any, 0, len(seriesMap))
		for series, count := range seriesMap {
			navigation = append(navigation, map[string]any{
				"title":         series,
				"href":          serverURL + basePath + "/v2/series/" + url.PathEscape(series),
				"type":          opds.MimeTypeOPDS2,
				"rel":           "subsection",
				"numberOfItems": count,
			})
		}
		if page.NextCursor != "" {
			hasMore = true
			links = append(links, map[string]any{
				"rel":  "next",
				"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(page.NextCursor),
				"type": opds.MimeTypeOPDS2,
			})
		}
	}

	meta := map[string]any{
		"title":        "Series",
		"itemsPerPage": len(navigation),
	}
	if !hasMore {
		meta["numberOfItems"] = len(navigation)
	}

	return map[string]any{
		"metadata":   meta,
		"links":      links,
		"navigation": navigation,
	}, nil
}

func (s *opdsService) GetOPDS2SeriesBooks(ctx context.Context, serverURL, basePath string, seriesName string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "ExcludeAudiobooks", "series", seriesName, "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	publications := make([]map[string]any, 0, len(visible))
	for _, book := range visible {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	selfPath := basePath + "/v2/series/" + url.PathEscape(seriesName)
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	cursor := nextCursor(books, q.Limit)
	if cursor != "" {
		links = append(links, map[string]any{
			"rel":  "next",
			"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(cursor),
			"type": opds.MimeTypeOPDS2,
		})
	}

	meta := map[string]any{
		"title":        "Series: " + seriesName,
		"itemsPerPage": len(publications),
	}
	if cursor == "" {
		meta["numberOfItems"] = len(publications)
	}

	return map[string]any{
		"metadata":     meta,
		"links":        links,
		"publications": publications,
	}, nil
}

func (s *opdsService) GetOPDS2TagsCatalog(ctx context.Context, serverURL, basePath string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	limit := q.Limit
	if limit <= 0 {
		limit = int64(constants.MaxPaginationLimit)
	}
	selfPath := basePath + "/v2/tags"
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	var navigation []map[string]any
	hasMore := false

	if s.metadataService != nil {
		res, err := s.metadataService.ListTags(ctx, &request.MetadataFacetDto{Limit: int(limit), Cursor: q.Cursor}, claims)
		if err != nil {
			return nil, err
		}
		if data, ok := res.Data.([]*response.MetadataCountResponse); ok {
			navigation = make([]map[string]any, 0, len(data))
			for _, item := range data {
				navigation = append(navigation, map[string]any{
					"title":         item.Name,
					"href":          serverURL + basePath + "/v2/tags/" + url.PathEscape(item.Name),
					"type":          opds.MimeTypeOPDS2,
					"rel":           "subsection",
					"numberOfItems": item.BookCount,
				})
			}
		}
		if res.Pagination != nil && res.Pagination.NextCursor != "" {
			hasMore = true
			links = append(links, map[string]any{
				"rel":  "next",
				"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(res.Pagination.NextCursor),
				"type": opds.MimeTypeOPDS2,
			})
		}
	} else {
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
		navigation = make([]map[string]any, 0, len(tagsMap))
		for tag, count := range tagsMap {
			navigation = append(navigation, map[string]any{
				"title":         tag,
				"href":          serverURL + basePath + "/v2/tags/" + url.PathEscape(tag),
				"type":          opds.MimeTypeOPDS2,
				"rel":           "subsection",
				"numberOfItems": count,
			})
		}
		if page.NextCursor != "" {
			hasMore = true
			links = append(links, map[string]any{
				"rel":  "next",
				"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(page.NextCursor),
				"type": opds.MimeTypeOPDS2,
			})
		}
	}

	meta := map[string]any{
		"title":        "Tags & Genres",
		"itemsPerPage": len(navigation),
	}
	if !hasMore {
		meta["numberOfItems"] = len(navigation)
	}

	return map[string]any{
		"metadata":   meta,
		"links":      links,
		"navigation": navigation,
	}, nil
}

func (s *opdsService) GetOPDS2TagBooks(ctx context.Context, serverURL, basePath string, tagName string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, nil, "", "", "ExcludeAudiobooks", "tag", tagName, "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	publications := make([]map[string]any, 0, len(visible))
	for _, book := range visible {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	selfPath := basePath + "/v2/tags/" + url.PathEscape(tagName)
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	cursor := nextCursor(books, q.Limit)
	if cursor != "" {
		links = append(links, map[string]any{
			"rel":  "next",
			"href": serverURL + selfPath + "?cursor=" + url.QueryEscape(cursor),
			"type": opds.MimeTypeOPDS2,
		})
	}

	meta := map[string]any{
		"title":        "Tag: " + tagName,
		"itemsPerPage": len(publications),
	}
	if cursor == "" {
		meta["numberOfItems"] = len(publications)
	}

	return map[string]any{
		"metadata":     meta,
		"links":        links,
		"publications": publications,
	}, nil
}

func (s *opdsService) GetOPDS2Search(ctx context.Context, serverURL, basePath string, query string, q request.OPDSPageDto, claims *response.JWTClaims) (map[string]any, error) {
	basePath = resolveBasePath(basePath)
	claims = resolveClaims(claims)
	books, err := s.books.SearchBooks(ctx, nil, &query, "", "", "ExcludeAudiobooks", "", "", "", q.Cursor, q.Limit, claims.UId)
	if err != nil {
		return nil, err
	}
	visible := s.filterVisible(ctx, books, claims)
	publications := make([]map[string]any, 0, len(visible))
	for _, book := range visible {
		publications = append(publications, s.bookToOPDS2Publication(ctx, book, serverURL, basePath, claims))
	}

	selfPath := basePath + "/v2/search?q=" + url.QueryEscape(query)
	links := []map[string]any{
		{"rel": "self", "href": serverURL + selfPath, "type": opds.MimeTypeOPDS2},
		{"rel": "start", "href": serverURL + basePath + "/v2/catalog", "type": opds.MimeTypeOPDS2},
	}
	cursor := nextCursor(books, q.Limit)
	if cursor != "" {
		links = append(links, map[string]any{
			"rel":  "next",
			"href": serverURL + selfPath + "&cursor=" + url.QueryEscape(cursor),
			"type": opds.MimeTypeOPDS2,
		})
	}

	meta := map[string]any{
		"title":        "Search Results for " + query,
		"itemsPerPage": len(publications),
	}
	if cursor == "" {
		meta["numberOfItems"] = len(publications)
	}

	return map[string]any{
		"metadata":     meta,
		"links":        links,
		"publications": publications,
	}, nil
}
