package services

import (
	"context"
	"fmt"
	"strconv"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
)

type VBookService interface {
	GetHomeSections(ctx context.Context) ([]*response.VBookHomeItem, error)
	GetGenres(ctx context.Context) ([]*response.VBookGenreItem, error)
	GetBooks(ctx context.Context, search *string, facet string, facetID string, page int, limit int) (*response.VBookBookListResponse, error)
	SearchBooks(ctx context.Context, query string, page int, limit int) (*response.VBookBookListResponse, error)
	GetBookDetail(ctx context.Context, bookID string) (*response.VBookBookDetailResponse, error)
	GetTOC(ctx context.Context, bookID string) ([]*response.VBookTOCItem, error)
	GetChapterContent(ctx context.Context, bookID string, chapterID string) (*response.VBookChapterContentResponse, error)
}

type vbookService struct {
	bookRepo     repositories.BookCatalogRepository
	metadataRepo repositories.BookMetadataRepository
	bookService  BookService
}

func NewVBookService(bookRepo repositories.BookCatalogRepository, metadataRepo repositories.BookMetadataRepository, bookService BookService) VBookService {
	return &vbookService{
		bookRepo:     bookRepo,
		metadataRepo: metadataRepo,
		bookService:  bookService,
	}
}

func (s *vbookService) GetHomeSections(ctx context.Context) ([]*response.VBookHomeItem, error) {
	return []*response.VBookHomeItem{
		{
			Title:  "Sách mới cập nhật",
			Input:  "/api/v1/vbook/books?sort=updated",
			Script: "gen.js",
		},
		{
			Title:  "Sách xem nhiều",
			Input:  "/api/v1/vbook/books?sort=hot",
			Script: "gen.js",
		},
		{
			Title:  "Mới thêm gần đây",
			Input:  "/api/v1/vbook/books?sort=created",
			Script: "gen.js",
		},
	}, nil
}

func (s *vbookService) GetGenres(ctx context.Context) ([]*response.VBookGenreItem, error) {
	return []*response.VBookGenreItem{
		{
			Title:  "Tất cả sách",
			Input:  "/api/v1/vbook/books",
			Script: "gen.js",
		},
		{
			Title:  "Sê-ri",
			Input:  "/api/v1/vbook/books?facet=series",
			Script: "gen.js",
		},
		{
			Title:  "Tác giả",
			Input:  "/api/v1/vbook/books?facet=authors",
			Script: "gen.js",
		},
		{
			Title:  "Thẻ / Nhãn",
			Input:  "/api/v1/vbook/books?facet=tags",
			Script: "gen.js",
		},
	}, nil
}

func (s *vbookService) GetBooks(ctx context.Context, search *string, facet string, facetID string, page int, limit int) (*response.VBookBookListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	books, err := s.bookService.SearchBooks(ctx, nil, search, "all", "", "", facet, facetID, nil, "", int64(limit*page))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load books")
	}

	start := (page - 1) * limit
	if start >= len(books) {
		return &response.VBookBookListResponse{List: []*response.VBookBookItem{}}, nil
	}

	end := start + limit
	var nextPage *string
	if end < len(books) {
		nextStr := strconv.Itoa(page + 1)
		nextPage = &nextStr
	} else {
		end = len(books)
	}

	slice := books[start:end]
	items := make([]*response.VBookBookItem, 0, len(slice))
	for _, b := range slice {
		author := "Chưa rõ"
		if b.AuthorName != nil && *b.AuthorName != "" {
			author = *b.AuthorName
		}

		desc := ""
		if b.Description != nil {
			desc = *b.Description
		}

		coverURL := fmt.Sprintf("/api/v1/books/%s/cover", b.ID)
		items = append(items, &response.VBookBookItem{
			Name:        b.Title,
			Link:        fmt.Sprintf("/api/v1/vbook/detail?id=%s", b.ID),
			Cover:       coverURL,
			Description: fmt.Sprintf("Tác giả: %s | %s", author, desc),
			Host:        "NovelHub",
		})
	}

	return &response.VBookBookListResponse{
		List: items,
		Next: nextPage,
	}, nil
}

func (s *vbookService) SearchBooks(ctx context.Context, query string, page int, limit int) (*response.VBookBookListResponse, error) {
	return s.GetBooks(ctx, &query, "", "", page, limit)
}

func (s *vbookService) GetBookDetail(ctx context.Context, bookID string) (*response.VBookBookDetailResponse, error) {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load book")
	}

	author := "Chưa rõ"
	if book.AuthorName != nil && *book.AuthorName != "" {
		author = *book.AuthorName
	}

	description := ""
	if book.Description != nil {
		description = *book.Description
	}

	chapters, _ := s.bookService.ListChapters(ctx, bookID)
	detailStr := fmt.Sprintf("Tác giả: %s | Tổng số chương: %d", author, len(chapters))

	return &response.VBookBookDetailResponse{
		Name:        book.Title,
		Cover:       fmt.Sprintf("/api/v1/books/%s/cover", book.ID),
		Author:      author,
		Description: description,
		Detail:      detailStr,
		Host:        "NovelHub",
		Ongoing:     false,
	}, nil
}

func (s *vbookService) GetTOC(ctx context.Context, bookID string) ([]*response.VBookTOCItem, error) {
	chapters, err := s.bookService.ListChapters(ctx, bookID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load table of contents")
	}

	items := make([]*response.VBookTOCItem, 0, len(chapters))
	for _, c := range chapters {
		title := c.Title
		if title == "" {
			title = fmt.Sprintf("Chương %d", c.ChapterIndex+1)
		}
		items = append(items, &response.VBookTOCItem{
			Name: title,
			URL:  fmt.Sprintf("/api/v1/vbook/chap?book_id=%s&chapter_id=%s", bookID, c.ID),
			Host: "NovelHub",
		})
	}
	return items, nil
}

func (s *vbookService) GetChapterContent(ctx context.Context, bookID string, chapterID string) (*response.VBookChapterContentResponse, error) {
	htmlContent, err := s.bookService.GetChapterHTML(ctx, bookID, chapterID, "")
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load chapter content")
	}

	return &response.VBookChapterContentResponse{
		Content: htmlContent,
	}, nil
}
