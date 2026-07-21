package services

import (
	"context"
	"fmt"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/opds"
	"time"
)

type OPDSService interface {
	GetRootCatalog(ctx context.Context, serverURL string) (*opds.Feed, error)
	GetRecentBooks(ctx context.Context, serverURL string, limit int64) (*opds.Feed, error)
}

type opdsService struct {
	bookRepo        repositories.BookDBRepository
	settingsService SettingsService
}

func NewOPDSService(bookRepo repositories.BookDBRepository, settingsService SettingsService) OPDSService {
	return &opdsService{
		bookRepo:        bookRepo,
		settingsService: settingsService,
	}
}

func (s *opdsService) GetRootCatalog(ctx context.Context, serverURL string) (*opds.Feed, error) {
	now := time.Now()
	feed := &opds.Feed{
		Xmlns:     opds.NamespaceAtom,
		XmlnsDc:   opds.NamespaceDc,
		XmlnsOpds: opds.NamespaceOpds,
		XmlnsOs:   opds.NamespaceOs,
		ID:        "novelhub:root",
		Title:     "NovelHub OPDS Catalog",
		Updated:   now,
		Links: []opds.Link{
			{Rel: "self", Href: fmt.Sprintf("%s/api/opds/v1", serverURL), Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "start", Href: fmt.Sprintf("%s/api/opds/v1", serverURL), Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
		Entries: []opds.Entry{
			{
				ID:      "novelhub:recent",
				Title:   "Recent Additions",
				Updated: now,
				Links: []opds.Link{
					{Rel: "subsection", Href: fmt.Sprintf("%s/api/opds/v1/recent", serverURL), Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
				},
				Content: "Recently added books",
			},
		},
	}
	return feed, nil
}

func (s *opdsService) GetRecentBooks(ctx context.Context, serverURL string, limit int64) (*opds.Feed, error) {
	books, err := s.bookRepo.SearchBooks(ctx, nil, nil, "", "", "", "", "", nil, limit)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	feed := &opds.Feed{
		Xmlns:     opds.NamespaceAtom,
		XmlnsDc:   opds.NamespaceDc,
		XmlnsOpds: opds.NamespaceOpds,
		ID:        "novelhub:recent",
		Title:     "Recent Additions",
		Updated:   now,
		Links: []opds.Link{
			{Rel: "self", Href: fmt.Sprintf("%s/api/opds/v1/recent", serverURL), Type: "application/atom+xml;profile=opds-catalog;kind=acquisition"},
			{Rel: "start", Href: fmt.Sprintf("%s/api/opds/v1", serverURL), Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
	}

	for _, book := range books {
		entry := s.bookToEntry(book, serverURL)
		feed.Entries = append(feed.Entries, entry)
	}

	return feed, nil
}

func (s *opdsService) bookToEntry(book *models.BookEntity, serverURL string) opds.Entry {
	summary := ""
	if book.Description != nil {
		summary = *book.Description
	}

	entry := opds.Entry{
		ID:      fmt.Sprintf("novelhub:book:%s", book.ID),
		Title:   book.Title,
		Updated: book.UpdatedAt,
		Summary: summary,
		Links: []opds.Link{
			{Rel: "http://opds-spec.org/image/thumbnail", Href: fmt.Sprintf("%s/api/v1/books/%s/cover", serverURL, book.ID), Type: "image/jpeg"},
			{Rel: "http://opds-spec.org/image", Href: fmt.Sprintf("%s/api/v1/books/%s/cover", serverURL, book.ID), Type: "image/jpeg"},
			{Rel: "http://opds-spec.org/acquisition", Href: fmt.Sprintf("%s/api/v1/books/%s/download", serverURL, book.ID), Type: "application/epub+zip"},
		},
	}

	if book.AuthorName != nil && *book.AuthorName != "" {
		entry.Author = &opds.Author{Name: *book.AuthorName}
	}

	return entry
}
