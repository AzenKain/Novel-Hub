package services

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type VBookService interface {
	GetHomeSections(ctx context.Context, baseURL string) ([]*response.VBookHomeItem, error)
	GetGenres(ctx context.Context, baseURL string) ([]*response.VBookGenreItem, error)
	GetBooks(ctx context.Context, baseURL string, search *string, sort string, facet string, facetID string, page int, limit int) (*response.VBookBookListResponse, error)
	SearchBooks(ctx context.Context, baseURL string, query string, page int, limit int) (*response.VBookBookListResponse, error)
	GetBookDetail(ctx context.Context, baseURL string, bookID string) (*response.VBookBookDetailResponse, error)
	GetTOC(ctx context.Context, baseURL string, bookID string) ([]*response.VBookTOCItem, error)
	GetChapterContent(ctx context.Context, bookID string, chapterID string) (*response.VBookChapterContentResponse, error)
	GetPluginJSON(ctx context.Context, baseURL string) (*response.VBookRegistryResponse, error)
	GetPluginZip(ctx context.Context, baseURL string) ([]byte, error)
}

type vbookService struct {
	bookRepo     repositories.BookCatalogRepository
	metadataRepo repositories.BookMetadataRepository
	bookService  BookService
	vbookFS      fs.FS
	cache        cache.Cache
}

func NewVBookService(bookRepo repositories.BookCatalogRepository, metadataRepo repositories.BookMetadataRepository, bookService BookService, vbookFS fs.FS, ramCache cache.Cache) VBookService {
	return &vbookService{
		bookRepo:     bookRepo,
		metadataRepo: metadataRepo,
		bookService:  bookService,
		vbookFS:      vbookFS,
		cache:        ramCache,
	}
}

func (s *vbookService) GetHomeSections(ctx context.Context, baseURL string) ([]*response.VBookHomeItem, error) {
	return []*response.VBookHomeItem{
		{
			Title:  "Sách mới cập nhật",
			Input:  baseURL + "/api/v1/vbook/books?sort=updated",
			Script: "gen.js",
		},
		{
			Title:  "Sách xem nhiều",
			Input:  baseURL + "/api/v1/vbook/books?sort=hot",
			Script: "gen.js",
		},
		{
			Title:  "Mới thêm gần đây",
			Input:  baseURL + "/api/v1/vbook/books?sort=created",
			Script: "gen.js",
		},
	}, nil
}

func (s *vbookService) GetGenres(ctx context.Context, baseURL string) ([]*response.VBookGenreItem, error) {
	return []*response.VBookGenreItem{
		{
			Title:  "Tất cả sách",
			Input:  baseURL + "/api/v1/vbook/books",
			Script: "gen.js",
		},
		{
			Title:  "Sê-ri",
			Input:  baseURL + "/api/v1/vbook/books?facet=series",
			Script: "gen.js",
		},
		{
			Title:  "Tác giả",
			Input:  baseURL + "/api/v1/vbook/books?facet=authors",
			Script: "gen.js",
		},
		{
			Title:  "Thẻ / Nhãn",
			Input:  baseURL + "/api/v1/vbook/books?facet=tags",
			Script: "gen.js",
		},
	}, nil
}

func (s *vbookService) GetBooks(ctx context.Context, baseURL string, search *string, sort string, facet string, facetID string, page int, limit int) (*response.VBookBookListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	nav := ""
	if sort == "hot" {
		nav = "hot"
	} else if facetID == "" {
		switch facet {
		case "series":
			nav = "series"
		case "authors":
			nav = "authors"
		case "tags":
			nav = "tags"
		}
	} else {
		switch facet {
		case "authors":
			facet = "author"
		case "tags":
			facet = "tag"
		}
	}

	books, err := s.bookService.SearchBooks(ctx, nil, search, nav, "", "", facet, facetID, nil, "", int64(limit*page+1))
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
	}
	if end > len(books) {
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

		cover := ""
		if b.CoverURL != nil {
			cover = *b.CoverURL
		}

		items = append(items, &response.VBookBookItem{
			Name:        b.Title,
			Link:        baseURL + "/api/v1/vbook/detail?id=" + b.ID,
			Cover:       cover,
			Description: fmt.Sprintf("Tác giả: %s | %s", author, desc),
			Host:        baseURL,
		})
	}

	return &response.VBookBookListResponse{
		List: items,
		Next: nextPage,
	}, nil
}

func (s *vbookService) SearchBooks(ctx context.Context, baseURL string, query string, page int, limit int) (*response.VBookBookListResponse, error) {
	return s.GetBooks(ctx, baseURL, &query, "", "", "", page, limit)
}

func (s *vbookService) GetBookDetail(ctx context.Context, baseURL string, bookID string) (*response.VBookBookDetailResponse, error) {
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

	cover := ""
	if book.CoverURL != nil {
		cover = *book.CoverURL
	}

	chapters, _ := s.bookService.ListChapters(ctx, bookID)
	detailStr := fmt.Sprintf("Tác giả: %s | Tổng số chương: %d", author, len(chapters))

	return &response.VBookBookDetailResponse{
		Name:        book.Title,
		Cover:       cover,
		Author:      author,
		Description: description,
		Detail:      detailStr,
		Host:        baseURL,
		Ongoing:     false,
	}, nil
}

func (s *vbookService) GetTOC(ctx context.Context, baseURL string, bookID string) ([]*response.VBookTOCItem, error) {
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
			URL:  baseURL + "/api/v1/vbook/chap?book_id=" + bookID + "&chapter_id=" + c.ID,
			Host: baseURL,
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

const vbookDescription = "Tiện ích đọc sách cá nhân tự lưu trữ từ máy chủ NovelHub của bạn"

func (s *vbookService) GetPluginJSON(ctx context.Context, baseURL string) (*response.VBookRegistryResponse, error) {
	return &response.VBookRegistryResponse{
		Metadata: response.VBookRegistryMetadata{
			Author:      "NovelHub",
			Description: vbookDescription,
		},
		Data: []*response.VBookEntryResponse{
			{
				Name:        "NovelHub",
				Author:      "NovelHub",
				Path:        baseURL + "/api/v1/vbook/plugin.zip",
				Lib:         baseURL + "/api/v1/vbook/plugin.json",
				Version:     2,
				Source:      baseURL,
				Icon:        baseURL + "/vbook/icon.png",
				Description: vbookDescription,
				Type:        "novel",
				Locale:      "vi_VN",
			},
		},
	}, nil
}

var vbookScripts = []string{"chap", "detail", "gen", "genre", "home", "search", "toc"}

func (s *vbookService) GetPluginZip(ctx context.Context, baseURL string) ([]byte, error) {
	if s.vbookFS == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "VBook assets are not available")
	}

	var cached []byte
	if s.cache != nil {
		key := fmt.Sprintf(constants.CacheKeyVBookPlugin, baseURL)
		if err := s.cache.GetOrFetch(ctx, key, &cached, 24*time.Hour, func() (any, error) {
			return s.buildPluginZip(ctx, baseURL)
		}); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}
	return s.buildPluginZip(ctx, baseURL)
}

func (s *vbookService) buildPluginZip(_ context.Context, baseURL string) ([]byte, error) {
	pluginManifest := &response.VBookPluginResponse{
		Metadata: response.VBookPluginMetadata{
			Name:        "NovelHub",
			Author:      "NovelHub",
			Version:     2,
			Source:      baseURL,
			Regexp:      ".*/api/v1/vbook/.*|.*/books/.*",
			Description: vbookDescription,
			Locale:      "vi_VN",
			Language:    "javascript",
			Type:        "novel",
		},
		Script: response.VBookPluginScript{
			Home:   "home.js",
			Genre:  "genre.js",
			Detail: "detail.js",
			Search: "search.js",
			Toc:    "toc.js",
			Chap:   "chap.js",
		},
	}
	pluginJSON, err := jsonx.Marshal(pluginManifest)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build VBook plugin")
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if w, err := zw.Create("plugin.json"); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build VBook plugin")
	} else if _, err := w.Write(pluginJSON); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build VBook plugin")
	}
	if iconData, err := fs.ReadFile(s.vbookFS, "icon.png"); err == nil {
		if w, err := zw.Create("icon.png"); err == nil {
			_, _ = w.Write(iconData)
		}
	}
	for _, name := range vbookScripts {
		data, err := fs.ReadFile(s.vbookFS, "src/"+name+".js")
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build VBook plugin")
		}
		script := strings.ReplaceAll(string(data), "{{BASE_URL}}", baseURL)
		w, err := zw.Create("src/" + name + ".js")
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build VBook plugin")
		}
		if _, err := w.Write([]byte(script)); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build VBook plugin")
		}
	}
	if err := zw.Close(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build VBook plugin")
	}
	return buf.Bytes(), nil
}
