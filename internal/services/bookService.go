package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"novelhub/pkg/database"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
)

var readerLinkAttrRegex = regexp.MustCompile(`(src|href)=["']([^"']+)["']`)
var styleBlockRegex = regexp.MustCompile(`(?i)(<style[^>]*>)([\s\S]*?)(</style>)`)

type BookService interface {
	GetBook(ctx context.Context, id string) (*models.BookEntity, error)
	SearchBooks(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, cursor *time.Time, limit int64) ([]*models.BookEntity, error)

	ListChapters(ctx context.Context, bookID string) ([]*models.ChapterEntity, error)
	GetBookFilePath(ctx context.Context, bookID string) (string, error)
	GetBookFile(ctx context.Context, bookID string, fileID string) (*models.BookFileEntity, error)
	ListBookFiles(ctx context.Context, bookID string) ([]*models.BookFileEntity, error)
	UploadBookFiles(ctx context.Context, bookID string, files []*multipart.FileHeader) (*models.BookFileUploadResult, error)
	ExtractMetadata(ctx context.Context, bookID string) error
	SearchDeep(ctx context.Context, query string, limit, offset int64) ([]*models.FTSResultEntity, error)
	GetDuplicates(ctx context.Context) ([]*models.DuplicateFileEntity, error)
	UpdateMetadata(ctx context.Context, bookID string, req *request.UpdateBookMetadataDto) error
	GetReaderBootstrap(ctx context.Context, bookID string, fileID string) (*models.ReaderBootstrapEntity, error)
	GetChapterHTML(ctx context.Context, bookID string, chapterID string, fileID string) (string, error)
	GetAsset(ctx context.Context, bookID string, assetPath string, fileID string) (*models.ReaderAssetEntity, error)
	ListImages(ctx context.Context, bookID string, fileID string) ([]string, error)
	UpdateCover(ctx context.Context, bookID string, input UpdateCoverInput) (string, error)
	ArchiveBook(ctx context.Context, id string, archived bool) error
	DeleteBook(ctx context.Context, id string) error

	CanReadBook(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool
	CanDownloadBook(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool
	FilterReadableBooks(ctx context.Context, books []*models.BookEntity, claims *response.JWTClaims) ([]*models.BookEntity, bool)
	SafeDownloadFilename(title string, ext string) string
}

type UpdateCoverInput struct {
	UploadedFileName string
	UploadedData     []byte
	CoverURL         string
	EPUBImagePath    string
}

type bookService struct {
	bookRepo    repositories.BookDBRepository
	fileRepo    repositories.BookFileRepository
	parsers     *bookparser.Registry
	txManager   database.TxManager
	settings    SettingsService
	permissions PermissionCache
}

func NewBookService(repo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, parsers *bookparser.Registry, txManager database.TxManager, settings SettingsService, permissions PermissionCache) BookService {
	return &bookService{
		bookRepo:    repo,
		fileRepo:    fileRepo,
		parsers:     parsers,
		txManager:   txManager,
		settings:    settings,
		permissions: permissions,
	}
}

func (s *bookService) GetBook(ctx context.Context, id string) (*models.BookEntity, error) {
	book, err := s.bookRepo.GetBook(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichBook(ctx, book)
	return book, nil
}

func (s *bookService) SearchBooks(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, cursor *time.Time, limit int64) ([]*models.BookEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	books, err := s.bookRepo.SearchBooks(ctx, libraryID, search, nav, collection, chip, facet, facetID, cursor, limit)
	if err != nil {
		return nil, err
	}
	s.enrichBooks(ctx, books)
	return books, nil
}

func (s *bookService) enrichBooks(ctx context.Context, books []*models.BookEntity) {
	if len(books) == 0 {
		return
	}

	bookIDs := make([]string, 0, len(books))
	authorIDMap := make(map[string]bool)
	authorIDs := make([]string, 0, len(books))

	for _, book := range books {
		if book == nil {
			continue
		}
		bookIDs = append(bookIDs, book.ID)
		if book.AuthorID != nil && *book.AuthorID != "" {
			if !authorIDMap[*book.AuthorID] {
				authorIDMap[*book.AuthorID] = true
				authorIDs = append(authorIDs, *book.AuthorID)
			}
		}
	}

	filesByBookID := make(map[string][]*models.BookFileEntity)
	if files, err := s.bookRepo.GetFilesByBookIDs(ctx, bookIDs); err == nil {
		for _, f := range files {
			filesByBookID[f.BookID] = append(filesByBookID[f.BookID], f)
		}
	}

	authorNameByID := make(map[string]string)
	if len(authorIDs) > 0 {
		if authors, err := s.bookRepo.GetAuthorsByIDs(ctx, authorIDs); err == nil {
			for _, a := range authors {
				if a != nil && a.Name != "" {
					authorNameByID[a.ID] = a.Name
				}
			}
		}
	}

	for _, book := range books {
		if book == nil {
			continue
		}
		if files, ok := filesByBookID[book.ID]; ok {
			book.Files = files
		}
		if book.AuthorID != nil && *book.AuthorID != "" {
			if name, ok := authorNameByID[*book.AuthorID]; ok {
				authorName := name
				book.AuthorName = &authorName
				continue
			}
		}
		if book.MetadataJSON == nil || *book.MetadataJSON == "" {
			continue
		}
		var meta struct {
			Creator  string   `json:"creator"`
			Creators []string `json:"creators"`
		}
		if err := jsonx.UnmarshalString(*book.MetadataJSON, &meta); err == nil {
			authorName := strings.TrimSpace(meta.Creator)
			if authorName == "" && len(meta.Creators) > 0 {
				authorName = strings.Join(meta.Creators, ", ")
			}
			if authorName != "" {
				book.AuthorName = &authorName
			}
		}
	}
}

func (s *bookService) enrichBook(ctx context.Context, book *models.BookEntity) {
	if book == nil {
		return
	}
	s.enrichBooks(ctx, []*models.BookEntity{book})
}

func (s *bookService) ListChapters(ctx context.Context, bookID string) ([]*models.ChapterEntity, error) {
	return s.bookRepo.ListChaptersByBook(ctx, bookID)
}

func (s *bookService) preferReadableFile(files []*models.BookFileEntity) *models.BookFileEntity {
	for _, file := range files {
		if s.isReadableFile(file) && strings.EqualFold(file.Format, "epub") {
			return file
		}
	}
	for _, file := range files {
		if s.isReadableFile(file) {
			return file
		}
	}
	return files[0]
}

func (s *bookService) isReadableFile(file *models.BookFileEntity) bool {
	if file == nil {
		return false
	}
	if s.parsers.HasFormat(file.Format) {
		return true
	}
	return s.parsers.HasPath(file.Path)
}

func (s *bookService) parserForFile(file *models.BookFileEntity) (bookparser.Parser, error) {
	if file == nil {
		return nil, fmt.Errorf("book file is nil")
	}
	return s.parsers.Parser(file.Format, file.Path)
}

func isAllowedBookFormat(ext string) bool {
	switch strings.ToLower(ext) {
	case ".epub", ".mobi", ".azw", ".azw3", ".amz", ".pdf", ".doc", ".docx", ".odt", ".txt", ".md", ".markdown", ".html", ".htm", ".rtf", ".fb2", ".fbz", ".zip", ".cbz", ".cbr", ".cbt", ".cb7":
		return true
	default:
		return false
	}
}

func (s *bookService) GetBookFilePath(ctx context.Context, bookID string) (string, error) {
	file, err := s.GetBookFile(ctx, bookID, "")
	if err != nil {
		return "", err
	}
	return file.Path, nil
}

func (s *bookService) GetBookFile(ctx context.Context, bookID string, fileID string) (*models.BookFileEntity, error) {
	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no files found for book %s", bookID)
	}
	if fileID != "" {
		for _, file := range files {
			if file.ID == fileID {
				return file, nil
			}
		}
		return nil, fmt.Errorf("file %s not found for book %s", fileID, bookID)
	}
	return s.preferReadableFile(files), nil
}

func (s *bookService) ListBookFiles(ctx context.Context, bookID string) ([]*models.BookFileEntity, error) {
	if _, err := s.bookRepo.GetBook(ctx, bookID); err != nil {
		return nil, err
	}
	return s.bookRepo.GetFilesByBookId(ctx, bookID)
}

func (s *bookService) UploadBookFiles(ctx context.Context, bookID string, files []*multipart.FileHeader) (*models.BookFileUploadResult, error) {
	if _, err := s.bookRepo.GetBook(ctx, bookID); err != nil {
		return nil, err
	}
	successCount := 0
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !isAllowedBookFormat(ext) {
			continue
		}
		src, err := file.Open()
		if err != nil {
			continue
		}
		saved, saveErr := s.fileRepo.SaveBook(ctx, bookID, file.Filename, src)
		closeErr := src.Close()
		if saveErr != nil || closeErr != nil {
			continue
		}
		fileID := uuid.Must(uuid.NewV7()).String()
		state := "managed"
		hash, _ := s.fileRepo.HashSHA256(ctx, saved.Path)
		hashPtr := &hash
		if hash == "" {
			hashPtr = nil
		}
		if err := s.bookRepo.CreateBookFile(ctx, sqlc.CreateBookFileParams{
			ID:        fileID,
			BookID:    bookID,
			Path:      saved.Path,
			Format:    saved.Format,
			SizeBytes: saved.SizeBytes,
			ModTime:   saved.ModTime,
			Hash:      convert.StrPtrToNullString(hashPtr),
			State:     convert.StrPtrToNullString(&state),
		}); err != nil {
			continue
		}
		successCount++
	}
	currentFiles, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return &models.BookFileUploadResult{Uploaded: successCount, Total: len(files), Files: currentFiles}, nil
}

func (s *bookService) ExtractMetadata(ctx context.Context, bookID string) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return err
	}

	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no files found for book %s", bookID)
	}

	file := s.preferReadableFile(files)
	filePath := file.Path
	meta := &bookparser.BookMetadata{}
	parser, parserErr := s.parserForFile(file)
	if parserErr == nil {
		parsed, err := parser.ParseMetadata(filePath)
		if err == nil && parsed != nil {
			meta = parsed
		}
		if len(meta.CoverData) == 0 {
			if coverData, coverType, err := fallbackCoverFromImages(parser, filePath); err == nil {
				meta.CoverData = coverData
				meta.CoverType = coverType
			}
		}
	}

	if meta.Title != "" {
		book.Title = meta.Title
	}
	if meta.Description != "" {
		book.Description = &meta.Description
	}
	if meta.MetadataJSON != "" {
		book.MetadataJSON = &meta.MetadataJSON
	} else if book.MetadataJSON == nil || *book.MetadataJSON == "" {
		fallbackJSON, _ := jsonx.MarshalString(map[string]string{
			"title":  book.Title,
			"format": file.Format,
		})
		book.MetadataJSON = &fallbackJSON
	}

	if len(meta.CoverData) > 0 {
		ext := coverExtFromContent(meta.CoverType, meta.CoverData)
		if coverURL, _, err := s.fileRepo.SaveCover(ctx, bookID, ext, meta.CoverData); err == nil {
			book.CoverURL = &coverURL
		}
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	txRepo := s.bookRepo.WithTx(tx)

	if meta.Author != "" {
		if authorID, err := ensureAuthor(ctx, txRepo, meta.Author); err == nil && authorID != "" {
			book.AuthorID = &authorID
		}
	}

	if parserErr == nil {
		spine, err := parser.ParseSpine(filePath)
		if err == nil {
			for _, ch := range spine {
				contentPath := ch.ContentPath
				chapter := &models.ChapterEntity{
					ID:           uuid.Must(uuid.NewV7()).String(),
					BookID:       bookID,
					Title:        ch.Title,
					ContentPath:  &contentPath,
					ChapterIndex: int64(ch.Index),
				}
				_ = txRepo.CreateChapter(ctx, chapter)
			}
		}
	}

	book.Status = "ready"
	if err := txRepo.UpdateBook(ctx, book); err != nil {
		return err
	}

	s.syncParsedMetadata(ctx, txRepo, book.ID, meta)

	return tx.Commit()
}

func (s *bookService) syncParsedMetadata(ctx context.Context, repo repositories.BookDBRepository, bookID string, meta *bookparser.BookMetadata) {
	if meta == nil {
		return
	}
	_ = repo.ClearBookSeries(ctx, bookID)
	if meta.Series != "" {
		series, err := repo.GetSeriesByName(ctx, meta.Series)
		if err != nil {
			series = &models.SeriesEntity{
				ID:   uuid.Must(uuid.NewV7()).String(),
				Name: meta.Series,
			}
			_ = repo.CreateSeries(ctx, series)
		}
		if series != nil && series.ID != "" {
			var index *string
			if meta.SeriesIndex != "" {
				index = &meta.SeriesIndex
			}
			_ = repo.LinkBookSeries(ctx, bookID, series.ID, index)
		}
	}

	_ = repo.ClearBookPublishers(ctx, bookID)
	if meta.Publisher != "" {
		publisher, err := repo.GetPublisherByName(ctx, meta.Publisher)
		if err != nil {
			publisher = &models.PublisherEntity{
				ID:   uuid.Must(uuid.NewV7()).String(),
				Name: meta.Publisher,
			}
			_ = repo.CreatePublisher(ctx, publisher)
		}
		if publisher != nil && publisher.ID != "" {
			_ = repo.LinkBookPublisher(ctx, bookID, publisher.ID)
		}
	}

	_ = repo.ClearBookLanguages(ctx, bookID)
	if meta.Language != "" {
		language, err := repo.GetLanguageByName(ctx, meta.Language)
		if err != nil {
			language = &models.LanguageEntity{
				ID:   uuid.Must(uuid.NewV7()).String(),
				Name: meta.Language,
			}
			_ = repo.CreateLanguage(ctx, language)
		}
		if language != nil && language.ID != "" {
			_ = repo.LinkBookLanguage(ctx, bookID, language.ID)
		}
	}

	_ = repo.ClearBookTags(ctx, bookID)
	for _, tagName := range meta.Subjects {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}
		tag, err := repo.GetTagByName(ctx, tagName)
		if err != nil {
			newTagID := uuid.Must(uuid.NewV7()).String()
			if err := repo.CreateTag(ctx, &models.TagEntity{ID: newTagID, Name: tagName}); err == nil {
				tag = &models.TagEntity{ID: newTagID, Name: tagName}
			}
		}
		if tag != nil && tag.ID != "" {
			_ = repo.AddBookTag(ctx, bookID, tag.ID)
		}
	}
}

func (s *bookService) SearchDeep(ctx context.Context, query string, limit, offset int64) ([]*models.FTSResultEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.bookRepo.SearchFTS(ctx, query, limit, offset)
}

func (s *bookService) GetDuplicates(ctx context.Context) ([]*models.DuplicateFileEntity, error) {
	return s.bookRepo.GetDuplicateFiles(ctx, 1000, 0)
}

func (s *bookService) UpdateMetadata(ctx context.Context, bookID string, req *request.UpdateBookMetadataDto) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return err
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	txRepo := s.bookRepo.WithTx(tx)

	var authorID *string
	if req.Author != "" {
		if id, err := ensureAuthor(ctx, txRepo, req.Author); err == nil && id != "" {
			authorID = &id
		}
	}

	book.Title = req.Title
	if req.Description != "" {
		book.Description = &req.Description
	} else {
		book.Description = nil
	}
	book.AuthorID = authorID

	if newJSON, err := mergeBookMetadataJSON(book.MetadataJSON, req); err == nil {
		book.MetadataJSON = &newJSON
	}

	err = txRepo.UpdateBook(ctx, book)
	if err != nil {
		return err
	}

	// Synchronize Series
	_ = txRepo.ClearBookSeries(ctx, book.ID)
	if req.Series != "" {
		series, err := txRepo.GetSeriesByName(ctx, req.Series)
		if err != nil {
			series = &models.SeriesEntity{
				ID:   uuid.Must(uuid.NewV7()).String(),
				Name: req.Series,
			}
			_ = txRepo.CreateSeries(ctx, series)
		}
		if series != nil && series.ID != "" {
			var seriesIndex *string
			if req.SeriesIndex != "" {
				seriesIndex = &req.SeriesIndex
			}
			_ = txRepo.LinkBookSeries(ctx, book.ID, series.ID, seriesIndex)
		}
	}

	// Synchronize Publisher
	_ = txRepo.ClearBookPublishers(ctx, book.ID)
	if req.Publisher != "" {
		publisher, err := txRepo.GetPublisherByName(ctx, req.Publisher)
		if err != nil {
			publisher = &models.PublisherEntity{
				ID:   uuid.Must(uuid.NewV7()).String(),
				Name: req.Publisher,
			}
			_ = txRepo.CreatePublisher(ctx, publisher)
		}
		if publisher != nil && publisher.ID != "" {
			_ = txRepo.LinkBookPublisher(ctx, book.ID, publisher.ID)
		}
	}

	// Synchronize Language
	_ = txRepo.ClearBookLanguages(ctx, book.ID)
	if req.Language != "" {
		language, err := txRepo.GetLanguageByName(ctx, req.Language)
		if err != nil {
			language = &models.LanguageEntity{
				ID:   uuid.Must(uuid.NewV7()).String(),
				Name: req.Language,
			}
			_ = txRepo.CreateLanguage(ctx, language)
		}
		if language != nil && language.ID != "" {
			_ = txRepo.LinkBookLanguage(ctx, book.ID, language.ID)
		}
	}

	// Synchronize Tags
	_ = txRepo.ClearBookTags(ctx, book.ID)
	for _, tagName := range req.Subjects {
		if tagName == "" {
			continue
		}
		tag, err := txRepo.GetTagByName(ctx, tagName)
		if err != nil {
			newTagID := uuid.Must(uuid.NewV7()).String()
			err = txRepo.CreateTag(ctx, &models.TagEntity{
				ID:   newTagID,
				Name: tagName,
			})
			if err == nil {
				tag = &models.TagEntity{ID: newTagID, Name: tagName}
			}
		}
		if tag != nil && tag.ID != "" {
			_ = txRepo.AddBookTag(ctx, book.ID, tag.ID)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil || len(files) == 0 {
		return nil // No file to update
	}
	file := s.preferReadableFile(files)
	parser, err := s.parserForFile(file)
	if err != nil {
		return nil
	}

	meta := &bookparser.BookMetadata{
		Title:       req.Title,
		Author:      req.Author,
		Description: req.Description,
	}

	if err := parser.SaveOriginalMetadataAndFix(file.Path, meta); err != nil {
		// Log error but don't fail the API since DB is updated
		fmt.Printf("Failed to update source metadata for %s: %v\n", bookID, err)
	}

	return nil
}

func (s *bookService) GetReaderBootstrap(ctx context.Context, bookID string, fileID string) (*models.ReaderBootstrapEntity, error) {
	book, err := s.GetBook(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if fileID != "" {
		file, err := s.GetBookFile(ctx, bookID, fileID)
		if err != nil {
			return nil, err
		}
		chapters, err := s.parseFileChapters(file)
		if err != nil {
			return nil, err
		}
		return &models.ReaderBootstrapEntity{Book: book, Chapters: chapters}, nil
	}
	chapters, err := s.ListChapters(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if len(chapters) == 0 {
		file, fileErr := s.GetBookFile(ctx, bookID, "")
		if fileErr == nil && s.isReadableFile(file) {
			if parsedChapters, parseErr := s.parseFileChapters(file); parseErr == nil {
				chapters = parsedChapters
			}
		}
	}
	return &models.ReaderBootstrapEntity{Book: book, Chapters: chapters}, nil
}

func (s *bookService) parseFileChapters(file *models.BookFileEntity) ([]*models.ChapterEntity, error) {
	parser, err := s.parserForFile(file)
	if err != nil {
		return nil, err
	}
	spine, err := parser.ParseSpine(file.Path)
	if err != nil {
		return nil, err
	}
	chapters := make([]*models.ChapterEntity, 0, len(spine))
	for _, ch := range spine {
		contentPath := ch.ContentPath
		chapters = append(chapters, &models.ChapterEntity{
			ID:           file.ID + ":" + strconv.Itoa(ch.Index),
			BookID:       file.BookID,
			Title:        ch.Title,
			ContentPath:  &contentPath,
			ChapterIndex: int64(ch.Index),
		})
	}
	return chapters, nil
}



func (s *bookService) GetChapterHTML(ctx context.Context, bookID string, chapterID string, fileID string) (string, error) {
	file, err := s.GetBookFile(ctx, bookID, fileID)
	if err != nil {
		return "", err
	}
	filePath := file.Path
	parser, err := s.parserForFile(file)
	if err != nil {
		return "", err
	}

	var contentPath string
	if fileID != "" {
		targetIndex, ok := fileChapterIndex(file.ID, chapterID)
		if !ok {
			return "", fmt.Errorf("chapter content path not found")
		}
		spine, err := parser.ParseSpine(filePath)
		if err != nil {
			return "", err
		}
		for _, ch := range spine {
			if ch.Index == targetIndex {
				contentPath = ch.ContentPath
				break
			}
		}
	} else {
		chapters, err := s.ListChapters(ctx, bookID)
		if err != nil {
			return "", err
		}
		for _, ch := range chapters {
			if ch.ID == chapterID {
				if ch.ContentPath != nil {
					contentPath = *ch.ContentPath
				}
				break
			}
		}
		if contentPath == "" && s.isReadableFile(file) {
			if index, ok := fileChapterIndex(file.ID, chapterID); ok {
				spine, err := parser.ParseSpine(filePath)
				if err == nil {
					for _, ch := range spine {
						if ch.Index == index {
							contentPath = ch.ContentPath
							break
						}
					}
				}
			}
		}
	}
	if contentPath == "" {
		return "", fmt.Errorf("chapter content path not found")
	}

	if contentPath == bookparser.RawFileContentPath {
		return rawFileReaderHTML(bookID, file.ID, file.Path), nil
	}

	content, err := parser.GetChapterContent(filePath, contentPath)
	if err != nil {
		return "", err
	}

	return rewriteReaderHTML(content, bookID, contentPath, fileID), nil
}



func (s *bookService) GetAsset(ctx context.Context, bookID string, assetPath string, fileID string) (*models.ReaderAssetEntity, error) {
	file, err := s.GetBookFile(ctx, bookID, fileID)
	if err != nil {
		return nil, err
	}
	parser, err := s.parserForFile(file)
	if err != nil {
		return nil, err
	}
	data, err := parser.GetAsset(file.Path, assetPath)
	if err != nil {
		return nil, err
	}

	contentType := readerAssetContentType(assetPath)
	if contentType == "text/css" {
		data = []byte(scopeReaderCSS(string(data)))
	}

	return &models.ReaderAssetEntity{
		Data:        data,
		ContentType: contentType,
	}, nil
}

func (s *bookService) ListImages(ctx context.Context, bookID string, fileID string) ([]string, error) {
	file, err := s.GetBookFile(ctx, bookID, fileID)
	if err != nil {
		return nil, err
	}
	parser, err := s.parserForFile(file)
	if err != nil {
		return []string{}, nil
	}
	return parser.ListImages(file.Path)
}

func (s *bookService) UpdateCover(ctx context.Context, bookID string, input UpdateCoverInput) (string, error) {
	if _, err := s.GetBook(ctx, bookID); err != nil {
		return "", err
	}

	coverData, ext, err := s.resolveCoverData(ctx, bookID, input)
	if err != nil {
		return "", err
	}

	coverURLPath, _, err := s.fileRepo.SaveCover(ctx, bookID, ext, coverData)
	if err != nil {
		return "", err
	}

	if err := s.updateCoverURL(ctx, bookID, coverURLPath); err != nil {
		return "", err
	}
	return coverURLPath, nil
}

func (s *bookService) FilterReadableBooks(ctx context.Context, books []*models.BookEntity, claims *response.JWTClaims) ([]*models.BookEntity, bool) {
	if len(books) == 0 {
		return books, true
	}
	if claims == nil {
		settings, err := s.settings.Public(ctx)
		if err == nil && settings.GuestAccess.Mode == "login_required" {
			return nil, false
		}
		out := make([]*models.BookEntity, 0, len(books))
		for _, book := range books {
			if book != nil && s.settings.GuestAllows(book.LibraryID) {
				out = append(out, book)
			}
		}
		return out, true
	}
	out := make([]*models.BookEntity, 0, len(books))
	for _, book := range books {
		if book != nil && s.CanReadBook(ctx, book, claims) {
			out = append(out, book)
		}
	}
	return out, true
}

func (s *bookService) CanReadBook(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	if book == nil {
		return false
	}
	if claims == nil {
		return s.settings.GuestAllows(book.LibraryID)
	}
	return s.permissions.CanRoles(claims.RoleIDs, claims.Roles, "book.read", map[string]any{"library_id": book.LibraryID})
}

func (s *bookService) CanDownloadBook(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	if book == nil {
		return false
	}
	if claims == nil {
		return false
	}
	admin := s.permissions.IsAdmin(claims.RoleIDs, claims.Roles)
	if !s.settings.PolicyAllows("download", book.LibraryID, admin) {
		return false
	}
	return s.permissions.CanRoles(claims.RoleIDs, claims.Roles, "book.download", map[string]any{"library_id": book.LibraryID})
}

func (s *bookService) SafeDownloadFilename(title string, ext string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		name = "book"
	}
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, name)
	name = strings.Trim(name, " .-")
	if name == "" {
		name = "book"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return name + ext
}

func (s *bookService) resolveCoverData(ctx context.Context, bookID string, input UpdateCoverInput) ([]byte, string, error) {
	if len(input.UploadedData) > 0 {
		ext := strings.ToLower(filepath.Ext(input.UploadedFileName))
		if ext == "" {
			ext = ".jpg"
		}
		return input.UploadedData, normalizeCoverExt(ext), nil
	}

	if input.EPUBImagePath != "" {
		file, err := s.GetBookFile(ctx, bookID, "")
		if err != nil {
			return nil, "", err
		}
		parser, err := s.parserForFile(file)
		if err != nil {
			return nil, "", err
		}
		coverData, err := parser.GetAsset(file.Path, input.EPUBImagePath)
		if err != nil {
			return nil, "", err
		}
		return coverData, normalizeCoverExt(filepath.Ext(input.EPUBImagePath)), nil
	}

	if input.CoverURL != "" {
		parsed, err := url.Parse(input.CoverURL)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cover URL")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, "", fmt.Errorf("cover URL must use http or https scheme")
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.CoverURL, nil)
		if err != nil {
			return nil, "", err
		}
		client := netx.NewSafeHTTPClient(15 * time.Second)
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("cover download blocked or failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, "", fmt.Errorf("cover download failed with status %d", resp.StatusCode)
		}
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		if ct != "" && !strings.HasPrefix(ct, "image/") {
			return nil, "", fmt.Errorf("cover URL did not return an image (got %s)", ct)
		}
		coverData, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
		if err != nil {
			return nil, "", err
		}
		return coverData, normalizeCoverExt(filepath.Ext(parsed.Path)), nil
	}

	return nil, "", fmt.Errorf("no cover provided")
}

func (s *bookService) updateCoverURL(ctx context.Context, bookID string, coverURL string) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return err
	}
	book.CoverURL = &coverURL
	if err := s.bookRepo.UpdateBook(ctx, book); err != nil {
		return err
	}
	return nil
}

func (s *bookService) ArchiveBook(ctx context.Context, id string, archived bool) error {
	book, err := s.bookRepo.GetBook(ctx, id)
	if err != nil {
		return err
	}
	if archived {
		book.Status = "archived"
	} else if book.Status == "archived" {
		book.Status = "ready"
	}
	return s.bookRepo.UpdateBook(ctx, book)
}

func (s *bookService) DeleteBook(ctx context.Context, id string) error {
	if err := s.fileRepo.RemoveBookDir(ctx, id); err != nil {
		log.Warn().Err(err).Str("book_id", id).Msg("failed to remove book files")
	}
	return s.bookRepo.DeleteBook(ctx, id)
}
