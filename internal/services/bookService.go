package services

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
)

var readerLinkAttrRegex = regexp.MustCompile(`(src|href)=["']([^"']+)["']`)
var styleBlockRegex = regexp.MustCompile(`(?i)(<style[^>]*>)([\s\S]*?)(</style>)`)

type BookService interface {
	GetBook(ctx context.Context, id string) (*models.BookEntity, error)
	SearchBooks(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, limit, offset int64) ([]*models.BookEntity, error)
	SearchBooksCursor(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, cursor string, limit int64) ([]*models.BookEntity, error)
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
}

type UpdateCoverInput struct {
	UploadedFileName string
	UploadedData     []byte
	CoverURL         string
	EPUBImagePath    string
}

type bookService struct {
	bookRepo repositories.BookDBRepository
	fileRepo repositories.BookFileRepository
	parsers  *bookparser.Registry
	db       *sql.DB
}

func NewBookService(repo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, parsers *bookparser.Registry, db *sql.DB) BookService {
	return &bookService{
		bookRepo: repo,
		fileRepo: fileRepo,
		parsers:  parsers,
		db:       db,
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

func (s *bookService) SearchBooks(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, limit, offset int64) ([]*models.BookEntity, error) {
	books, err := s.bookRepo.SearchBooks(ctx, libraryID, search, nav, collection, chip, facet, facetID, limit, offset)
	if err != nil {
		return nil, err
	}
	s.enrichBooks(ctx, books)
	return books, nil
}

func (s *bookService) SearchBooksCursor(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, cursor string, limit int64) ([]*models.BookEntity, error) {
	// Default to current time if cursor is empty
	cursorTime := time.Now()
	if cursor != "" {
		if t, err := time.Parse(time.RFC3339Nano, cursor); err == nil {
			cursorTime = t
		}
	}
	list, err := s.bookRepo.SearchBooksCursor(ctx, libraryID, search, nav, collection, chip, facet, facetID, cursorTime, limit)
	if err != nil {
		return nil, err
	}
	s.enrichBooks(ctx, list)
	return list, nil
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
		if err := sonic.UnmarshalString(*book.MetadataJSON, &meta); err == nil {
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
		if err := s.bookRepo.CreateBookFile(ctx, repositories.BookFileRecordParams{
			ID:        fileID,
			BookID:    bookID,
			Path:      saved.Path,
			Format:    saved.Format,
			SizeBytes: saved.SizeBytes,
			ModTime:   saved.ModTime,
			Hash:      hashPtr,
			State:     &state,
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
		fallbackJSON, _ := sonic.MarshalString(map[string]string{
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

	tx, err := s.db.BeginTx(ctx, nil)
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

func ensureAuthor(ctx context.Context, repo repositories.BookDBRepository, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	author, err := repo.GetAuthorByName(ctx, name)
	if err == nil && author != nil {
		return author.ID, nil
	}
	newAuthorID := uuid.Must(uuid.NewV7()).String()
	if err := repo.CreateAuthor(ctx, &models.AuthorEntity{
		ID:   newAuthorID,
		Name: name,
	}); err != nil {
		return "", err
	}
	return newAuthorID, nil
}

func mergeBookMetadataJSON(existing *string, req *request.UpdateBookMetadataDto) (string, error) {
	metaMap := map[string]interface{}{}
	if existing != nil && strings.TrimSpace(*existing) != "" {
		_ = sonic.UnmarshalString(*existing, &metaMap)
	}
	setStringMetadata(metaMap, "title", req.Title)
	setStringMetadata(metaMap, "creator", req.Author)
	setStringMetadata(metaMap, "description", req.Description)
	setStringMetadata(metaMap, "publisher", req.Publisher)
	setStringMetadata(metaMap, "language", req.Language)
	setStringMetadata(metaMap, "date", req.Date)
	setStringMetadata(metaMap, "series", req.Series)
	setStringMetadata(metaMap, "seriesIndex", req.SeriesIndex)
	if len(req.Subjects) > 0 {
		metaMap["subject"] = req.Subjects
	} else {
		delete(metaMap, "subject")
	}

	rawMeta, _ := metaMap["meta"].([]interface{})
	rawMeta = upsertMetaValue(rawMeta, "calibre:series", req.Series)
	rawMeta = upsertMetaValue(rawMeta, "calibre:series_index", req.SeriesIndex)
	if len(rawMeta) > 0 {
		metaMap["meta"] = rawMeta
	} else {
		delete(metaMap, "meta")
	}
	return sonic.MarshalString(metaMap)
}

func setStringMetadata(metaMap map[string]interface{}, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		delete(metaMap, key)
		return
	}
	metaMap[key] = value
}

func upsertMetaValue(rawMeta []interface{}, name string, value string) []interface{} {
	value = strings.TrimSpace(value)
	for index, item := range rawMeta {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		itemName, _ := m["name"].(string)
		if itemName == "" {
			itemName, _ = m["Name"].(string)
		}
		if itemName != name {
			continue
		}
		if value == "" {
			return append(rawMeta[:index], rawMeta[index+1:]...)
		}
		m["name"] = name
		m["content"] = value
		return rawMeta
	}
	if value == "" {
		return rawMeta
	}
	return append(rawMeta, map[string]interface{}{"name": name, "content": value})
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
	return s.bookRepo.SearchFTS(ctx, query, limit, offset)
}

func (s *bookService) GetDuplicates(ctx context.Context) ([]*models.DuplicateFileEntity, error) {
	return s.bookRepo.GetDuplicateFiles(ctx)
}

func (s *bookService) UpdateMetadata(ctx context.Context, bookID string, req *request.UpdateBookMetadataDto) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
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
			ID:           fileChapterID(file.ID, ch.Index),
			BookID:       file.BookID,
			Title:        ch.Title,
			ContentPath:  &contentPath,
			ChapterIndex: int64(ch.Index),
		})
	}
	return chapters, nil
}

func fileChapterID(fileID string, index int) string {
	return fileID + ":" + strconv.Itoa(index)
}

func fileChapterIndex(fileID string, chapterID string) (int, bool) {
	prefix := fileID + ":"
	if !strings.HasPrefix(chapterID, prefix) {
		return 0, false
	}
	index, err := strconv.Atoi(strings.TrimPrefix(chapterID, prefix))
	return index, err == nil
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

func rawFileReaderHTML(bookID string, fileID string, filePath string) string {
	sourceURL := `/api/v1/reader/` + url.PathEscape(bookID) + `/file?file_id=` + url.QueryEscape(fileID)
	title := html.EscapeString(bookparser.TitleFromPath(filePath))
	return `<div class="novelhub-raw-reader" style="width: 100%; height: 100%; margin: 0; padding: 0; overflow: hidden;"><iframe title="` + title + `" src="` + sourceURL + `" style="width: 100%; height: 100%; border: 0; background: #fff;" loading="eager"></iframe></div>`
}

func rewriteReaderHTML(content string, bookID string, contentPath string, fileID string) string {
	baseDir := filepath.ToSlash(filepath.Dir(contentPath))
	if baseDir == "." {
		baseDir = ""
	}

	rewritten := readerLinkAttrRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := readerLinkAttrRegex.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		attr := matches[1]
		value := matches[2]
		if strings.HasPrefix(value, "http://") ||
			strings.HasPrefix(value, "https://") ||
			strings.HasPrefix(value, "data:") ||
			strings.HasPrefix(value, "#") {
			return match
		}

		resolved := filepath.ToSlash(filepath.Join(baseDir, value))
		assetURL := `/api/v1/reader/` + url.PathEscape(bookID) + `/asset/` + escapeAssetPath(resolved)
		if fileID != "" {
			assetURL += `?file_id=` + url.QueryEscape(fileID)
		}
		return attr + `="` + assetURL + `"`
	})

	// Scope inline CSS inside <style> tags to avoid bleeding layout styles into the global document
	rewritten = styleBlockRegex.ReplaceAllStringFunc(rewritten, func(match string) string {
		submatches := styleBlockRegex.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}
		rewrittenCSS := scopeReaderCSS(submatches[2])
		return submatches[1] + rewrittenCSS + submatches[3]
	})

	return rewritten
}

func escapeAssetPath(assetPath string) string {
	parts := strings.Split(filepath.ToSlash(assetPath), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
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

func scopeReaderCSS(css string) string {
	var out strings.Builder
	for i := 0; i < len(css); {
		openRel := strings.IndexByte(css[i:], '{')
		if openRel < 0 {
			out.WriteString(css[i:])
			break
		}
		open := i + openRel
		close := matchingCSSBrace(css, open)
		if close < 0 {
			out.WriteString(css[i:])
			break
		}

		selector := css[i:open]
		selectorTrimmed := strings.TrimSpace(selector)
		out.WriteString(scopeReaderSelectorList(selector))
		out.WriteByte('{')
		block := css[open+1 : close]
		if readerCSSAtRuleScopesChildren(selectorTrimmed) {
			out.WriteString(scopeReaderCSS(block))
		} else {
			out.WriteString(block)
		}
		out.WriteByte('}')
		i = close + 1
	}
	return out.String()
}

func matchingCSSBrace(css string, open int) int {
	depth := 0
	for i := open; i < len(css); i++ {
		switch css[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func scopeReaderSelectorList(selector string) string {
	trimmed := strings.TrimSpace(selector)
	if strings.HasPrefix(trimmed, "@") {
		return selector
	}
	parts := strings.Split(selector, ",")
	for i, part := range parts {
		parts[i] = scopeReaderSelector(part)
	}
	return strings.Join(parts, ",")
}

func scopeReaderSelector(selector string) string {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return selector
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, ".reader-content") {
		return trimmed
	}
	for _, prefix := range []string{"body", "html"} {
		if lower == prefix {
			return ".reader-content"
		}
		if strings.HasPrefix(lower, prefix+" ") {
			return ".reader-content " + strings.TrimSpace(trimmed[len(prefix):])
		}
		if strings.HasPrefix(lower, prefix+">") {
			return ".reader-content " + strings.TrimSpace(trimmed[len(prefix):])
		}
		if strings.HasPrefix(lower, prefix+".") || strings.HasPrefix(lower, prefix+"#") || strings.HasPrefix(lower, prefix+":") {
			restStart := len(prefix)
			for restStart < len(trimmed) && trimmed[restStart] != ' ' && trimmed[restStart] != '>' && trimmed[restStart] != '+' && trimmed[restStart] != '~' {
				restStart++
			}
			if restStart < len(trimmed) {
				return ".reader-content " + strings.TrimSpace(trimmed[restStart:])
			}
			return ".reader-content"
		}
	}
	return ".reader-content " + trimmed
}

func readerCSSAtRuleScopesChildren(selector string) bool {
	lower := strings.ToLower(strings.TrimSpace(selector))
	return strings.HasPrefix(lower, "@media") ||
		strings.HasPrefix(lower, "@supports") ||
		strings.HasPrefix(lower, "@container") ||
		strings.HasPrefix(lower, "@layer") ||
		strings.HasPrefix(lower, "@scope")
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
		if isPrivateHost(parsed.Hostname()) {
			return nil, "", fmt.Errorf("cover URL must not point to a private address")
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.CoverURL, nil)
		if err != nil {
			return nil, "", err
		}
		client := http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", err
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

func normalizeCoverExt(ext string) string {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if ext == ".jpeg" {
		return ".jpg"
	}
	switch ext {
	case ".jpg", ".png", ".webp", ".gif", ".bmp":
		return ext
	default:
		return ".jpg"
	}
}

func fallbackCoverFromImages(parser bookparser.Parser, filePath string) ([]byte, string, error) {
	images, err := parser.ListImages(filePath)
	if err != nil {
		return nil, "", err
	}
	for _, imagePath := range images {
		data, err := parser.GetAsset(filePath, imagePath)
		if err != nil || len(data) == 0 {
			continue
		}
		contentType := readerAssetContentType(imagePath)
		if !isSupportedCoverContentType(contentType) {
			contentType = http.DetectContentType(data)
		}
		if isSupportedCoverContentType(contentType) {
			return data, contentType, nil
		}
	}
	return nil, "", fmt.Errorf("no supported image cover found")
}

func coverExtFromContent(contentType string, data []byte) string {
	contentType = strings.ToLower(contentType)
	if !isSupportedCoverContentType(contentType) {
		contentType = strings.ToLower(http.DetectContentType(data))
	}
	switch {
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	case strings.Contains(contentType, "bmp"):
		return ".bmp"
	default:
		return ".jpg"
	}
}

func isSupportedCoverContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "jpeg") ||
		strings.Contains(contentType, "jpg") ||
		strings.Contains(contentType, "png") ||
		strings.Contains(contentType, "webp") ||
		strings.Contains(contentType, "gif") ||
		strings.Contains(contentType, "bmp")
}

func readerAssetContentType(assetPath string) string {
	switch strings.ToLower(filepath.Ext(assetPath)) {
	case ".css":
		return "text/css"
	case ".html", ".xhtml":
		return "text/html"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".avif":
		return "image/avif"
	case ".svg":
		return "image/svg+xml"
	case ".js":
		return "application/javascript"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
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
	// 1. Remove book directory containing files/covers from filesystem
	if err := s.fileRepo.RemoveBookDir(ctx, id); err != nil {
		log.Warn().Err(err).Str("book_id", id).Msg("failed to remove book files")
	}
	// 2. Remove book from database
	return s.bookRepo.DeleteBook(ctx, id)
}

func isPrivateHost(host string) bool {
	if host == "" {
		return true
	}
	lower := strings.ToLower(host)
	if lower == "localhost" {
		return true
	}
	ips, err := net.LookupHost(host)
	if err != nil {
		// If we can't resolve, allow it — the HTTP client will fail anyway.
		return false
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
		// Block metadata endpoint IPs (e.g., AWS 169.254.169.254)
		if ip.Equal(net.IPv4(169, 254, 169, 254)) {
			return true
		}
	}
	return false
}
