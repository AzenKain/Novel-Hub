package services

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type VBookService interface {
	GetHomeSections(ctx context.Context, baseURL string) ([]*response.VBookHomeItem, error)
	GetGenres(ctx context.Context, baseURL string) ([]*response.VBookGenreItem, error)
	GetBooks(ctx context.Context, baseURL string, search *string, sort string, facet string, facetID string, pageStr string, limit int, claims *response.JWTClaims) (*response.VBookBookListResponse, error)
	SearchBooks(ctx context.Context, baseURL string, query string, pageStr string, limit int, claims *response.JWTClaims) (*response.VBookBookListResponse, error)
	GetBookDetail(ctx context.Context, baseURL string, bookID string, claims *response.JWTClaims) (*response.VBookBookDetailResponse, error)
	GetTOC(ctx context.Context, baseURL string, bookID string, claims *response.JWTClaims) ([]*response.VBookTOCItem, error)
	GetChapterContent(ctx context.Context, bookID string, chapterID string, claims *response.JWTClaims) (*response.VBookChapterContentResponse, error)
	GetAudioBooks(ctx context.Context, baseURL string, pageStr string, limit int, claims *response.JWTClaims) (*response.VBookBookListResponse, error)
	GetAudioPlaylist(ctx context.Context, bookID string, claims *response.JWTClaims) ([]*response.VBookAudioTrack, error)
	ResolveAudioStream(ctx context.Context, bookID string, fileID string, claims *response.JWTClaims) (*models.BookFileEntity, error)
	GetPluginJSON(ctx context.Context, baseURL string) (*response.VBookRegistryResponse, error)
	GetPluginZip(ctx context.Context, baseURL string) ([]byte, error)
	GetPluginZipAudio(ctx context.Context, baseURL string) ([]byte, error)
}

type vbookService struct {
	bookRepo     repositories.BookCatalogRepository
	metadataRepo repositories.BookMetadataRepository
	audiobookRepo repositories.AudiobookRepository
	bookService  BookService
	vbookFS      fs.FS
	cache        cache.Cache
}

func NewVBookService(bookRepo repositories.BookCatalogRepository, metadataRepo repositories.BookMetadataRepository, audiobookRepo repositories.AudiobookRepository, bookService BookService, vbookFS fs.FS, ramCache cache.Cache) VBookService {
	return &vbookService{
		bookRepo:      bookRepo,
		metadataRepo:  metadataRepo,
		audiobookRepo: audiobookRepo,
		bookService:   bookService,
		vbookFS:       vbookFS,
		cache:         ramCache,
	}
}

func (s *vbookService) GetHomeSections(ctx context.Context, baseURL string) ([]*response.VBookHomeItem, error) {
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

func (s *vbookService) GetGenres(ctx context.Context, baseURL string) ([]*response.VBookGenreItem, error) {
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

func (s *vbookService) GetBooks(ctx context.Context, baseURL string, search *string, sort string, facet string, facetID string, pageStr string, limit int, claims *response.JWTClaims) (*response.VBookBookListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	c := resolveClaims(claims)
	userID := ""
	if !isGuestClaims(c) {
		userID = c.UId
	}

	var cursorTime *time.Time
	var cursorID string

	// If pageStr is a cursor (contains '|')
	if strings.Contains(pageStr, "|") {
		parts := strings.SplitN(pageStr, "|", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorTime = &t
				cursorID = parts[1]
			}
		}
	} else if pageStr != "" && pageStr != "1" {
		// Fallback: if it's a page number greater than 1, we can calculate offset
		if p, err := strconv.Atoi(pageStr); err == nil && p > 1 {
			nav := ""
			if sort == "hot" {
				nav = "hot"
			} else if sort == "random" {
				nav = "random"
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
			books, err := s.bookService.SearchBooks(ctx, nil, search, nav, "", "ExcludeAudiobooks", facet, facetID, sort, "", int64(limit*p+1), userID)
			if err != nil {
				return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load books")
			}
			filtered, allowed := s.bookService.FilterReadableBooks(ctx, books, claims)
			if !allowed {
				return &response.VBookBookListResponse{List: []*response.VBookBookItem{}}, nil
			}
			start := (p - 1) * limit
			if start >= len(filtered) {
				return &response.VBookBookListResponse{List: []*response.VBookBookItem{}}, nil
			}
			end := start + limit
			var nextPage *string
			if end < len(filtered) {
				nextStr := strconv.Itoa(p + 1)
				nextPage = &nextStr
			}
			if end > len(filtered) {
				end = len(filtered)
			}
			slice := filtered[start:end]
			items := mapBooksToItems(slice, baseURL)
			return &response.VBookBookListResponse{
				List: items,
				Next: nextPage,
			}, nil
		}
	}

	nav := ""
	if sort == "hot" {
		nav = "hot"
	} else if sort == "random" {
		nav = "random"
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

	cursorStr := ""
	if cursorTime != nil {
		cursorStr = cursorTime.Format(time.RFC3339Nano) + "|" + cursorID
	}
	books, err := s.bookService.SearchBooks(ctx, nil, search, nav, "", "ExcludeAudiobooks", facet, facetID, sort, cursorStr, int64(limit+1), userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load books")
	}
	filtered, allowed := s.bookService.FilterReadableBooks(ctx, books, claims)
	if !allowed {
		return &response.VBookBookListResponse{List: []*response.VBookBookItem{}}, nil
	}

	var nextPage *string
	end := limit
	if len(filtered) < end {
		end = len(filtered)
	}
	if sort != "random" && len(filtered) > limit {
		lastBook := filtered[limit-1]
		nextStr := lastBook.CreatedAt.Format(time.RFC3339Nano) + "|" + lastBook.ID
		nextPage = &nextStr
	}

	slice := filtered[:end]
	items := mapBooksToItems(slice, baseURL)

	return &response.VBookBookListResponse{
		List: items,
		Next: nextPage,
	}, nil
}

func mapBooksToItems(books []*models.BookEntity, baseURL string) []*response.VBookBookItem {
	items := make([]*response.VBookBookItem, 0, len(books))
	for _, b := range books {
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
			Link:        "/api/v1/vbook/detail?id=" + b.ID,
			Cover:       cover,
			Description: fmt.Sprintf("Tác giả: %s | %s", author, desc),
			Host:        baseURL,
		})
	}
	return items
}

func (s *vbookService) GetAudioBooks(ctx context.Context, baseURL string, pageStr string, limit int, claims *response.JWTClaims) (*response.VBookBookListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// vbook pages by cursor "time|id" like GetBooks; no offset paging.
	var cursorTime *time.Time
	var cursorID string
	if strings.Contains(pageStr, "|") {
		parts := strings.SplitN(pageStr, "|", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorTime = &t
				cursorID = parts[1]
			}
		}
	}

	ids, err := s.audiobookRepo.ListBooksWithAudio(ctx, cursorTime, cursorID, int64(limit+1))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load audio books")
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, ids)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load audio books")
	}

	// GetBooksByIDs is cache-backed and not order-preserving; re-sort by ids.
	byID := make(map[string]*models.BookEntity, len(books))
	for _, b := range books {
		byID[b.ID] = b
	}
	ordered := make([]*models.BookEntity, 0, len(ids))
	for _, id := range ids {
		if b, ok := byID[id]; ok {
			ordered = append(ordered, b)
		}
	}

	filtered, _ := s.bookService.FilterReadableBooks(ctx, ordered, claims)

	end := limit
	if len(filtered) < end {
		end = len(filtered)
	}
	slice := filtered[:end]

	var nextPage *string
	if len(filtered) > limit && len(slice) > 0 {
		last := slice[len(slice)-1]
		nextStr := last.UpdatedAt.Format(time.RFC3339Nano) + "|" + last.ID
		nextPage = &nextStr
	}

	return &response.VBookBookListResponse{
		List: mapBooksToItems(slice, baseURL),
		Next: nextPage,
	}, nil
}

func (s *vbookService) GetAudioPlaylist(ctx context.Context, bookID string, claims *response.JWTClaims) ([]*response.VBookAudioTrack, error) {
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load book")
	}
	if !s.bookService.CanReadBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
	}

	chapters, err := s.audiobookRepo.ListChapters(ctx, bookID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load audio playlist")
	}

	var tracks []*response.VBookAudioTrack
	var curFileID string
	var runTitles []string
	flush := func() {
		if curFileID == "" || len(runTitles) == 0 {
			return
		}
		name := runTitles[0]
		desc := ""
		if len(runTitles) > 1 {
			name += fmt.Sprintf(" (+%d)", len(runTitles)-1)
			desc = fmt.Sprintf("%d chương", len(runTitles))
		}
		tracks = append(tracks, &response.VBookAudioTrack{
			Name:        name,
			URL:         "/api/v1/vbook/audio/stream?book_id=" + url.QueryEscape(bookID) + "&file_id=" + url.QueryEscape(curFileID),
			Description: desc,
		})
	}
	for _, c := range chapters {
		if c.FileID == nil || *c.FileID == "" {
			continue
		}
		fid := *c.FileID
		if fid != curFileID {
			flush()
			curFileID = fid
			runTitles = runTitles[:0]
		}
		title := c.Title
		if title == "" {
			title = fmt.Sprintf("Chương %d", c.ChapterIndex+1)
		}
		runTitles = append(runTitles, title)
	}
	flush()

	// Fallback: if no audiobook_chapters exist, build tracks from audio book_files directly.
	if len(tracks) == 0 {
		files, err := s.bookService.ListBookFiles(ctx, bookID)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load audio files")
		}
		for i, f := range files {
			if !isAudioFormat(f.Format) {
				continue
			}
			name := strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path))
			if name == "" {
				name = fmt.Sprintf("Track %d", i+1)
			}
			tracks = append(tracks, &response.VBookAudioTrack{
				Name: name,
				URL:  "/api/v1/vbook/audio/stream?book_id=" + url.QueryEscape(bookID) + "&file_id=" + url.QueryEscape(f.ID),
			})
		}
	}

	if len(tracks) == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "No audio tracks")
	}
	return tracks, nil
}

func (s *vbookService) ResolveAudioStream(ctx context.Context, bookID string, fileID string, claims *response.JWTClaims) (*models.BookFileEntity, error) {
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load book")
	}
	if !s.bookService.CanReadBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "You do not have access to this book")
	}

	file, err := s.bookService.GetBookFile(ctx, bookID, fileID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Audio file not found")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load audio file")
	}
	return file, nil
}

func (s *vbookService) SearchBooks(ctx context.Context, baseURL string, query string, pageStr string, limit int, claims *response.JWTClaims) (*response.VBookBookListResponse, error) {
	return s.GetBooks(ctx, baseURL, &query, "", "", "", pageStr, limit, claims)
}

func (s *vbookService) GetBookDetail(ctx context.Context, baseURL string, bookID string, claims *response.JWTClaims) (*response.VBookBookDetailResponse, error) {
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load book")
	}
	if !s.bookService.CanReadBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
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

func (s *vbookService) GetTOC(ctx context.Context, baseURL string, bookID string, claims *response.JWTClaims) ([]*response.VBookTOCItem, error) {
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load book")
	}
	if !s.bookService.CanReadBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
	}

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
			URL:  "/api/v1/vbook/chap?book_id=" + bookID + "&chapter_id=" + c.ID,
			Host: baseURL,
		})
	}
	return items, nil
}

func (s *vbookService) GetChapterContent(ctx context.Context, bookID string, chapterID string, claims *response.JWTClaims) (*response.VBookChapterContentResponse, error) {
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load book")
	}
	if !s.bookService.CanReadBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
	}

	htmlContent, err := s.bookService.GetChapterHTML(ctx, bookID, chapterID, "")
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load chapter content")
	}

	return &response.VBookChapterContentResponse{
		Content: htmlContent,
	}, nil
}

const vbookDescription = "Tiện ích đọc sách cá nhân tự lưu trữ từ máy chủ NovelHub của bạn"
const vbookAudioDescription = "Nghe audiobook tự lưu trữ từ máy chủ NovelHub của bạn"

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
			{
				Name:        "NovelHub Audio",
				Author:      "NovelHub",
				Path:        baseURL + "/api/v1/vbook/plugin-audio.zip",
				Lib:         baseURL + "/api/v1/vbook/plugin.json",
				Version:     3,
				Source:      baseURL,
				Icon:        baseURL + "/vbook/icon.png",
				Description: vbookAudioDescription,
				Type:        "audio",
				Locale:      "vi_VN",
			},
		},
	}, nil
}

var vbookScripts = []string{"chap", "detail", "gen", "home", "search", "toc"}
var audioScripts = map[string]string{
	"home":   "audio_home",
	"gen":    "gen",
	"search": "search",
	"detail": "audio_detail",
	"toc":    "audio_toc",
	"chap":   "audio_chap",
	"track":  "track",
}

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

// ponytail: audio ext shares the same fs; zip cache key suffixed -audio
// so the two plugin variants don't clobber each other.
func (s *vbookService) GetPluginZipAudio(ctx context.Context, baseURL string) ([]byte, error) {
	if s.vbookFS == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "VBook assets are not available")
	}

	var cached []byte
	if s.cache != nil {
		key := fmt.Sprintf(constants.CacheKeyVBookPlugin+"-audio", baseURL)
		if err := s.cache.GetOrFetch(ctx, key, &cached, 24*time.Hour, func() (any, error) {
			return s.buildAudioPluginZip(ctx, baseURL)
		}); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}
	return s.buildAudioPluginZip(ctx, baseURL)
}

func (s *vbookService) buildPluginZip(_ context.Context, baseURL string) ([]byte, error) {
	return zipVBookPlugin(s.vbookFS, baseURL,
		&response.VBookPluginResponse{
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
				Detail: "detail.js",
				Search: "search.js",
				Toc:    "toc.js",
				Chap:   "chap.js",
			},
		},
		vbookFiles())
}

func (s *vbookService) buildAudioPluginZip(_ context.Context, baseURL string) ([]byte, error) {
	return zipVBookPlugin(s.vbookFS, baseURL,
		&response.VBookPluginResponse{
			Metadata: response.VBookPluginMetadata{
				Name:        "NovelHub Audio",
				Author:      "NovelHub",
				Version:     3,
				Source:      baseURL,
				Regexp:      ".*/api/v1/vbook/.*|.*/books/.*",
				Description: vbookAudioDescription,
				Locale:      "vi_VN",
				Language:    "javascript",
				Type:        "audio",
			},
			Script: response.VBookPluginScript{
				Home:   "home.js",
				Detail: "detail.js",
				Search: "search.js",
				Toc:    "toc.js",
				Chap:   "chap.js",
				Track:  "track.js",
			},
		},
		audioScripts)
}

// vbookFiles maps zip src/ name -> disk src/ name (identity for the novel pack).
func vbookFiles() map[string]string {
	out := make(map[string]string, len(vbookScripts))
	for _, name := range vbookScripts {
		out[name] = name
	}
	return out
}

// zipVBookPlugin writes plugin.json + icon + all script files into a zip.
// srcFiles maps the zip entry base name -> disk src/<name>.js.
func zipVBookPlugin(vbookFS fs.FS, baseURL string, manifest *response.VBookPluginResponse, srcFiles map[string]string) ([]byte, error) {
	pluginJSON, err := jsonx.Marshal(manifest)
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
	if iconData, err := fs.ReadFile(vbookFS, "icon.png"); err == nil {
		if w, err := zw.Create("icon.png"); err == nil {
			_, _ = w.Write(iconData)
		}
	}
	for dest, src := range srcFiles {
		data, err := fs.ReadFile(vbookFS, "src/"+src+".js")
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to build VBook plugin")
		}
		script := strings.ReplaceAll(string(data), "{{BASE_URL}}", baseURL)
		w, err := zw.Create("src/" + dest + ".js")
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

func isAudioFormat(format string) bool {
	switch strings.ToLower(format) {
	case "mp3", "m4a", "m4b", "flac", "ogg", "opus", "wav", "aac":
		return true
	default:
		return false
	}
}
