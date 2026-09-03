package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/calibre"
	"novelhub/pkg/jsonx"
)

type CalibreServerService interface {
	GetLibraryInfo(ctx context.Context, claims *response.JWTClaims) (*response.CalibreLibraryInfoResponse, error)
	GetCategories(ctx context.Context, libraryID string, claims *response.JWTClaims) (map[string]*response.CalibreCategorySummary, error)
	GetCategory(ctx context.Context, libraryID string, encodedCategory string, num int64, offset int64, sort string, sortOrder string, claims *response.JWTClaims) (*response.CalibreCategoryDetailResponse, error)
	GetBooksInCategory(ctx context.Context, libraryID string, encodedCategory string, encodedItem string, num int64, offset int64, sort string, sortOrder string, claims *response.JWTClaims) (*response.CalibreBooksInResponse, error)
	SearchBooks(ctx context.Context, libraryID string, query string, num int64, offset int64, sort string, sortOrder string, claims *response.JWTClaims) (*response.CalibreSearchResponse, error)
	GetBooksMetadata(ctx context.Context, libraryID string, bookIDs []string, claims *response.JWTClaims) (map[string]*response.CalibreBookMetadataResponse, error)
	GetBookMetadata(ctx context.Context, libraryID string, bookID string, claims *response.JWTClaims) (*response.CalibreBookMetadataResponse, error)
	GetBookCover(ctx context.Context, bookID string, thumb bool, claims *response.JWTClaims) (string, error)
	GetBookFile(ctx context.Context, bookID string, format string, claims *response.JWTClaims) (filePath string, filename string, err error)
}

type calibreServerService struct {
	bookRepo        repositories.BookDBRepository
	diskRepo        repositories.BookFileRepository
	bookService     BookService
	metadataService MetadataService
	libraryService  LibraryService
	booksDir        string
}

func NewCalibreServerService(
	bookRepo repositories.BookDBRepository,
	diskRepo repositories.BookFileRepository,
	bookService BookService,
	metadataService MetadataService,
	libraryService LibraryService,
	booksDir string,
) CalibreServerService {
	if strings.TrimSpace(booksDir) == "" {
		booksDir = filepath.Join(".", "data", "books")
	}
	return &calibreServerService{
		bookRepo:        bookRepo,
		diskRepo:        diskRepo,
		bookService:     bookService,
		metadataService: metadataService,
		libraryService:  libraryService,
		booksDir:        booksDir,
	}
}

func (s *calibreServerService) GetLibraryInfo(ctx context.Context, claims *response.JWTClaims) (*response.CalibreLibraryInfoResponse, error) {
	libraryIDs, err := s.libraryService.ReadableLibraryIDs(ctx, claims)
	if err != nil {
		return nil, err
	}

	libraryMap := make(map[string]string)
	defaultLibrary := "1"

	if len(libraryIDs) == 0 {
		libraryMap["1"] = "NovelHub Library"
	} else {
		for i, id := range libraryIDs {
			lib, err := s.libraryService.GetLibrary(ctx, id, claims)
			if err == nil && lib != nil {
				libraryMap[id] = lib.Name
			} else {
				libraryMap[id] = "Library " + id
			}
			if i == 0 {
				defaultLibrary = id
			}
		}
	}

	return &response.CalibreLibraryInfoResponse{
		LibraryMap:     libraryMap,
		DefaultLibrary: defaultLibrary,
	}, nil
}

func (s *calibreServerService) GetCategories(ctx context.Context, libraryID string, claims *response.JWTClaims) (map[string]*response.CalibreCategorySummary, error) {
	totalBooks := int64(0)
	if libraryID != "" && libraryID != "None" {
		if ids, err := s.bookRepo.ListBookIDsByLibrary(ctx, libraryID, 100000); err == nil {
			totalBooks = int64(len(ids))
		}
	} else {
		readableIDs, err := s.libraryService.ReadableLibraryIDs(ctx, claims)
		if err == nil {
			for _, libID := range readableIDs {
				if ids, err := s.bookRepo.ListBookIDsByLibrary(ctx, libID, 100000); err == nil {
					totalBooks += int64(len(ids))
				}
			}
		}
	}

	q := &request.MetadataFacetDto{Limit: 1}
	authorRes, _ := s.metadataService.ListAuthors(ctx, q, claims)
	seriesRes, _ := s.metadataService.ListSeries(ctx, q, claims)
	tagsRes, _ := s.metadataService.ListTags(ctx, q, claims)
	formatsRes, _ := s.metadataService.ListFormats(ctx, q, claims)
	publishersRes, _ := s.metadataService.ListPublishers(ctx, q, claims)

	libSuffix := ""
	if libraryID != "" && libraryID != "None" {
		libSuffix = "/" + libraryID
	}

	var authorCount, seriesCount, tagsCount, formatsCount, publishersCount int64
	if authorRes != nil && authorRes.Pagination != nil {
		authorCount = authorRes.Pagination.TotalRecords
	}
	if seriesRes != nil && seriesRes.Pagination != nil {
		seriesCount = seriesRes.Pagination.TotalRecords
	}
	if tagsRes != nil && tagsRes.Pagination != nil {
		tagsCount = tagsRes.Pagination.TotalRecords
	}
	if formatsRes != nil && formatsRes.Pagination != nil {
		formatsCount = formatsRes.Pagination.TotalRecords
	}
	if publishersRes != nil && publishersRes.Pagination != nil {
		publishersCount = publishersRes.Pagination.TotalRecords
	}

	categories := map[string]*response.CalibreCategorySummary{
		"allbooks": {
			Name:       "All books",
			URL:        "/calibre/ajax/books_in/" + calibre.EncodeName("allbooks") + "/" + calibre.EncodeName("all") + libSuffix,
			IsCategory: false,
			Count:      totalBooks,
		},
		"authors": {
			Name:       "Authors",
			URL:        "/calibre/ajax/category/" + calibre.EncodeName("authors") + libSuffix,
			IsCategory: true,
			Count:      authorCount,
		},
		"series": {
			Name:       "Series",
			URL:        "/calibre/ajax/category/" + calibre.EncodeName("series") + libSuffix,
			IsCategory: true,
			Count:      seriesCount,
		},
		"tags": {
			Name:       "Tags",
			URL:        "/calibre/ajax/category/" + calibre.EncodeName("tags") + libSuffix,
			IsCategory: true,
			Count:      tagsCount,
		},
		"formats": {
			Name:       "Formats",
			URL:        "/calibre/ajax/category/" + calibre.EncodeName("formats") + libSuffix,
			IsCategory: true,
			Count:      formatsCount,
		},
		"publishers": {
			Name:       "Publishers",
			URL:        "/calibre/ajax/category/" + calibre.EncodeName("publishers") + libSuffix,
			IsCategory: true,
			Count:      publishersCount,
		},
	}

	return categories, nil
}

func (s *calibreServerService) GetCategory(ctx context.Context, libraryID string, encodedCategory string, num int64, offset int64, sort string, sortOrder string, claims *response.JWTClaims) (*response.CalibreCategoryDetailResponse, error) {
	categoryName := strings.ToLower(calibre.DecodeName(encodedCategory))
	if num <= 0 || num > 1000 {
		num = 100
	}
	if offset < 0 {
		offset = 0
	}
	if sort == "" {
		sort = "name"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}

	q := &request.MetadataFacetDto{
		Limit: int(num),
	}

	var res *response.PaginatedResponse
	var err error

	switch categoryName {
	case "authors":
		res, err = s.metadataService.ListAuthors(ctx, q, claims)
	case "series":
		res, err = s.metadataService.ListSeries(ctx, q, claims)
	case "tags":
		res, err = s.metadataService.ListTags(ctx, q, claims)
	case "formats":
		res, err = s.metadataService.ListFormats(ctx, q, claims)
	case "publishers":
		res, err = s.metadataService.ListPublishers(ctx, q, claims)
	default:
		return nil, apperrors.New(apperrors.ErrNotFound, "Category not found")
	}

	if err != nil {
		return nil, err
	}

	libSuffix := ""
	if libraryID != "" && libraryID != "None" {
		libSuffix = "/" + libraryID
	}

	items := make([]response.CalibreCategoryItem, 0)
	if res != nil && res.Data != nil {
		if rawItems, ok := res.Data.([]*response.MetadataCountResponse); ok {
			for _, item := range rawItems {
				itemID := item.ID
				if itemID == "" {
					itemID = item.Name
				}
				items = append(items, response.CalibreCategoryItem{
					Name:        item.Name,
					Count:       item.BookCount,
					URL:         "/calibre/ajax/books_in/" + calibre.EncodeName(categoryName) + "/" + calibre.EncodeName(itemID) + libSuffix,
					HasChildren: false,
				})
			}
		}
	}

	var total int64
	if res != nil && res.Pagination != nil {
		total = res.Pagination.TotalRecords
	}

	return &response.CalibreCategoryDetailResponse{
		CategoryName:  categoryName,
		BaseURL:       "/calibre/ajax/category/" + encodedCategory + libSuffix,
		TotalNum:      total,
		Offset:        offset,
		Num:           num,
		Sort:          sort,
		SortOrder:     sortOrder,
		Subcategories: []any{},
		Items:         items,
	}, nil
}

func (s *calibreServerService) GetBooksInCategory(ctx context.Context, libraryID string, encodedCategory string, encodedItem string, num int64, offset int64, sort string, sortOrder string, claims *response.JWTClaims) (*response.CalibreBooksInResponse, error) {
	categoryName := strings.ToLower(calibre.DecodeName(encodedCategory))
	itemName := calibre.DecodeName(encodedItem)

	if num <= 0 || num > 1000 {
		num = 100
	}
	if offset < 0 {
		offset = 0
	}

	userID := ""
	if claims != nil {
		userID = claims.UId
	}

	var libIDPtr *string
	if libraryID != "" && libraryID != "None" {
		libIDPtr = &libraryID
	}

	var facet, facetID string
	if categoryName != "allbooks" && categoryName != "newest" {
		facet = categoryName
		facetID = itemName

		switch categoryName {
		case "authors", "author":
			facet = "author"
			if author, err := s.bookRepo.GetAuthorByName(ctx, itemName); err == nil && author != nil {
				facetID = author.ID
			}
		case "series":
			facet = "series"
			if series, err := s.bookRepo.GetSeriesByName(ctx, itemName); err == nil && series != nil {
				facetID = series.ID
			}
		case "tags", "tag":
			facet = "tag"
			if tag, err := s.bookRepo.GetTagByName(ctx, itemName); err == nil && tag != nil {
				facetID = tag.ID
			}
		case "publishers", "publisher":
			facet = "publisher"
			if pub, err := s.bookRepo.GetPublisherByName(ctx, itemName); err == nil && pub != nil {
				facetID = pub.ID
			}
		case "languages", "language":
			facet = "language"
			if lang, err := s.bookRepo.GetLanguageByName(ctx, itemName); err == nil && lang != nil {
				facetID = lang.ID
			}
		case "formats", "format":
			facet = "format"
		}
	}

	books, err := s.bookRepo.SearchBooks(ctx, libIDPtr, nil, "", "", "", facet, facetID, sort, "", num, userID)
	if err != nil {
		return nil, err
	}

	filtered, _ := s.bookService.FilterReadableBooks(ctx, books, claims)
	bookIDs := make([]string, 0, len(filtered))
	for _, b := range filtered {
		bookIDs = append(bookIDs, b.ID)
	}

	return &response.CalibreBooksInResponse{
		TotalNum:  int64(len(bookIDs)),
		SortOrder: sortOrder,
		Offset:    offset,
		Num:       num,
		Sort:      sort,
		BookIDs:   bookIDs,
	}, nil
}

func (s *calibreServerService) SearchBooks(ctx context.Context, libraryID string, query string, num int64, offset int64, sort string, sortOrder string, claims *response.JWTClaims) (*response.CalibreSearchResponse, error) {
	if num <= 0 || num > 1000 {
		num = 100
	}
	if offset < 0 {
		offset = 0
	}

	userID := ""
	if claims != nil {
		userID = claims.UId
	}

	var libIDPtr *string
	if libraryID != "" && libraryID != "None" {
		libIDPtr = &libraryID
	}

	trimmedQuery := strings.TrimSpace(query)
	var searchPtr *string
	if trimmedQuery != "" {
		searchPtr = &trimmedQuery
	}

	books, err := s.bookRepo.SearchBooks(ctx, libIDPtr, searchPtr, "", "", "", "", "", sort, "", num, userID)
	if err != nil {
		return nil, err
	}

	filtered, _ := s.bookService.FilterReadableBooks(ctx, books, claims)
	bookIDs := make([]string, 0, len(filtered))
	for _, b := range filtered {
		bookIDs = append(bookIDs, b.ID)
	}

	return &response.CalibreSearchResponse{
		TotalNum:              int64(len(bookIDs)),
		SortOrder:             sortOrder,
		NumBooksWithoutSearch: int64(len(bookIDs)),
		Offset:                offset,
		Num:                   num,
		Sort:                  sort,
		BaseURL:               "/calibre/ajax/search",
		Query:                 query,
		LibraryID:             libraryID,
		BookIDs:               bookIDs,
	}, nil
}

func (s *calibreServerService) GetBooksMetadata(ctx context.Context, libraryID string, bookIDs []string, claims *response.JWTClaims) (map[string]*response.CalibreBookMetadataResponse, error) {
	if len(bookIDs) == 0 {
		return map[string]*response.CalibreBookMetadataResponse{}, nil
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, bookIDs)
	if err != nil {
		return nil, err
	}

	filtered, _ := s.bookService.FilterReadableBooks(ctx, books, claims)
	result := make(map[string]*response.CalibreBookMetadataResponse, len(filtered))

	for _, book := range filtered {
		result[book.ID] = s.entityToCalibreMetadata(book)
	}

	return result, nil
}

func (s *calibreServerService) GetBookMetadata(ctx context.Context, libraryID string, bookID string, claims *response.JWTClaims) (*response.CalibreBookMetadataResponse, error) {
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil {
		return nil, err
	}

	if !s.bookService.CanReadBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Access denied")
	}

	return s.entityToCalibreMetadata(book), nil
}

func (s *calibreServerService) entityToCalibreMetadata(book *models.BookEntity) *response.CalibreBookMetadataResponse {
	authors := make([]string, 0)
	if book.AuthorName != nil && strings.TrimSpace(*book.AuthorName) != "" {
		authors = append(authors, *book.AuthorName)
	}

	var seriesName *string
	seriesIndex := float64(1)
	var publisherName *string
	tags := make([]string, 0)
	languages := make([]string, 0)

	if book.MetadataJSON != nil && strings.TrimSpace(*book.MetadataJSON) != "" {
		var meta map[string]any
		if err := jsonx.UnmarshalString(*book.MetadataJSON, &meta); err == nil {
			if len(authors) == 0 {
				if creator, ok := meta["creator"].(string); ok && creator != "" {
					authors = append(authors, creator)
				}
			}
			if sName, ok := meta["series"].(string); ok && sName != "" {
				seriesName = &sName
			}
			if sIdx, ok := meta["seriesIndex"].(float64); ok {
				seriesIndex = sIdx
			} else if sIdxStr, ok := meta["seriesIndex"].(string); ok {
				if f, err := strconv.ParseFloat(sIdxStr, 64); err == nil {
					seriesIndex = f
				}
			}
			if pub, ok := meta["publisher"].(string); ok && pub != "" {
				publisherName = &pub
			}
			if lang, ok := meta["language"].(string); ok && lang != "" {
				languages = append(languages, lang)
			}
			if subjects, ok := meta["subject"].([]any); ok {
				for _, sub := range subjects {
					if subStr, ok := sub.(string); ok && subStr != "" {
						tags = append(tags, subStr)
					}
				}
			}
		}
	}

	if len(authors) == 0 {
		authors = []string{"Unknown"}
	}

	formats := make([]string, 0, len(book.Files))
	mainFormat := make(map[string]string)
	otherFormats := make(map[string]string)

	for i, f := range book.Files {
		if f != nil && f.Format != "" {
			fmtLower := strings.ToLower(f.Format)
			formats = append(formats, strings.ToUpper(fmtLower))
			downloadURL := fmt.Sprintf("/calibre/get/%s/%s", fmtLower, book.ID)
			if i == 0 {
				mainFormat[fmtLower] = downloadURL
			} else {
				otherFormats[fmtLower] = downloadURL
			}
		}
	}

	coverURL := fmt.Sprintf("/calibre/get/cover/%s", book.ID)
	thumbURL := fmt.Sprintf("/calibre/get/thumb/%s", book.ID)

	identifiers := make(map[string]string)
	identifiers["uuid"] = book.ID
	if book.GoogleBooksID != nil && *book.GoogleBooksID != "" {
		identifiers["google"] = *book.GoogleBooksID
	}
	if book.OpenLibraryID != nil && *book.OpenLibraryID != "" {
		identifiers["openlibrary"] = *book.OpenLibraryID
	}

	authorSort := strings.Join(authors, " & ")

	comments := ""
	if book.Description != nil {
		comments = *book.Description
	}

	return &response.CalibreBookMetadataResponse{
		Title:        book.Title,
		Authors:      authors,
		AuthorSort:   authorSort,
		Series:       seriesName,
		SeriesIndex:  seriesIndex,
		Rating:       0,
		Tags:         tags,
		Comments:     comments,
		Publisher:    publisherName,
		Pubdate:      book.CreatedAt.Format(time.RFC3339),
		Timestamp:    book.CreatedAt.Format(time.RFC3339),
		LastModified: book.UpdatedAt.Format(time.RFC3339),
		Identifiers:  identifiers,
		Languages:    languages,
		Cover:        coverURL,
		Thumbnail:    thumbURL,
		Formats:      formats,
		MainFormat:   mainFormat,
		OtherFormats: otherFormats,
	}
}

func (s *calibreServerService) GetBookCover(ctx context.Context, bookID string, thumb bool, claims *response.JWTClaims) (string, error) {
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil {
		return "", err
	}

	if !s.bookService.CanReadBook(ctx, book, claims) {
		return "", apperrors.New(apperrors.ErrForbidden, "Access denied")
	}

	if book.CoverURL == nil || strings.TrimSpace(*book.CoverURL) == "" {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}

	resolved, err := s.diskRepo.ResolveCoverPath(ctx, book.ID, *book.CoverURL)
	if err != nil {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}

	if _, err := os.Stat(resolved); err != nil {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover file missing on disk")
	}

	return resolved, nil
}

func (s *calibreServerService) GetBookFile(ctx context.Context, bookID string, format string, claims *response.JWTClaims) (filePath string, filename string, err error) {
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil {
		return "", "", err
	}

	formatLower := strings.ToLower(format)
	var targetFileID string
	for _, f := range book.Files {
		if f == nil {
			continue
		}
		if formatLower == "" || formatLower == "any" || strings.ToLower(f.Format) == formatLower {
			targetFileID = f.ID
			break
		}
	}

	if targetFileID == "" {
		return "", "", apperrors.New(apperrors.ErrNotFound, fmt.Sprintf("No %s format for this book", format))
	}

	return s.bookService.GetBookFileForDownload(ctx, bookID, targetFileID, claims)
}
