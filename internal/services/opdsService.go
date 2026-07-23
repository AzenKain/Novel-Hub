package services

import (
	"context"
	"fmt"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/constants"
	"novelhub/pkg/opds"
)

type OPDSService interface {
	GetRootCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error)
	GetRecentBooks(ctx context.Context, serverURL string, limit int64, claims *response.JWTClaims) (*opds.Feed, error)
	GetOPDS2Catalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (map[string]any, error)
}

type opdsService struct {
	books       BookService
	permissions PermissionCache
}

func NewOPDSService(books BookService, permissions PermissionCache) OPDSService {
	return &opdsService{books: books, permissions: permissions}
}

func (s *opdsService) GetRootCatalog(ctx context.Context, serverURL string, claims *response.JWTClaims) (*opds.Feed, error) {
	now := time.Now()
	return &opds.Feed{
		Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs,
		ID: "novelhub:root", Title: "NovelHub OPDS Catalog", Updated: now,
		Links:   []opds.Link{{Rel: "self", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"}, {Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"}},
		Entries: []opds.Entry{{ID: "novelhub:recent", Title: "Recent Additions", Updated: now, Links: []opds.Link{{Rel: "subsection", Href: serverURL + "/api/opds/v1/recent", Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"}}, Content: "Recently added books"}},
	}, nil
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
	feed := &opds.Feed{Xmlns: opds.NamespaceAtom, XmlnsDc: opds.NamespaceDc, XmlnsOpds: opds.NamespaceOpds, XmlnsOs: opds.NamespaceOs, ID: "novelhub:recent", Title: "Recent Additions", Updated: now, Links: []opds.Link{{Rel: "self", Href: serverURL + "/api/opds/v1/recent", Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"}, {Rel: "start", Href: serverURL + "/api/opds/v1", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"}}}
	for _, book := range books {
		feed.Entries = append(feed.Entries, s.bookToEntry(ctx, book, serverURL, claims))
	}
	return feed, nil
}

func (s *opdsService) bookToEntry(ctx context.Context, book *models.BookEntity, serverURL string, claims *response.JWTClaims) opds.Entry {
	summary := ""
	if book.Description != nil {
		summary = *book.Description
	}
	entry := opds.Entry{ID: "novelhub:book:" + book.ID, Title: book.Title, Updated: book.UpdatedAt, Summary: summary, Links: []opds.Link{{Rel: "http://opds-spec.org/image/thumbnail", Href: fmt.Sprintf("%s/api/v1/books/%s/cover", serverURL, book.ID), Type: "image/jpeg"}, {Rel: "http://opds-spec.org/image", Href: fmt.Sprintf("%s/api/v1/books/%s/cover", serverURL, book.ID), Type: "image/jpeg"}}}
	if s.canAcquire(ctx, book, claims) {
		entry.Links = append(entry.Links, opds.Link{Rel: "http://opds-spec.org/acquisition", Href: fmt.Sprintf("%s/api/v1/books/%s/download", serverURL, book.ID), Type: "application/epub+zip"})
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
			links = append(links, map[string]any{"rel": "http://opds-spec.org/acquisition", "href": fmt.Sprintf("%s/api/v1/books/%s/download", serverURL, book.ID), "type": "application/epub+zip"})
		}
		metadata := map[string]any{"title": book.Title, "identifier": "urn:novelhub:book:" + book.ID}
		if book.AuthorName != nil && *book.AuthorName != "" {
			metadata["author"] = *book.AuthorName
		}
		publications = append(publications, map[string]any{"metadata": metadata, "links": links})
	}
	return map[string]any{"metadata": map[string]any{"title": "NovelHub OPDS 2.0 Catalog"}, "links": []map[string]any{{"rel": "self", "href": serverURL + "/api/opds/v2/catalog", "type": "application/opds+json"}}, "publications": publications}, nil
}
