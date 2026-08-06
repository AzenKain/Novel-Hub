package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/calibre"
	"novelhub/pkg/config"
	"novelhub/pkg/convert"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/localfs"
)

type CalibreSyncService interface {
	ImportCalibreLibrary(ctx context.Context, calibreDirPath string, libraryID string) (int, error)
}

type calibreSyncService struct {
	bookRepo  repositories.BookDBRepository
	fileRepo  repositories.BookFileRepository
	txManager database.TxManager
}

func NewCalibreSyncService(bookRepo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, txManager database.TxManager) CalibreSyncService {
	return &calibreSyncService{
		bookRepo:  bookRepo,
		fileRepo:  fileRepo,
		txManager: txManager,
	}
}

func calibreRoot() string {
	if configured := strings.TrimSpace(config.GetConfigWithDefault("CALIBRE_IMPORT_DIR", "")); configured != "" {
		return configured
	}
	return filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "calibre")
}

func resolveCalibreDir(requested string) (string, error) {
	root, err := filepath.Abs(calibreRoot())
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Invalid Calibre import root")
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return root, nil
	}
	if filepath.IsAbs(requested) {
		rel, relErr := filepath.Rel(root, requested)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", apperrors.New(apperrors.ErrBadRequest,
				"Calibre path must be inside "+root+" (set CALIBRE_IMPORT_DIR to import from elsewhere)")
		}
		requested = rel
	}
	dir, err := localfs.SafeJoin(root, requested)
	if err != nil {
		return "", apperrors.New(apperrors.ErrBadRequest, "Invalid Calibre library path")
	}
	return dir, nil
}

func (s *calibreSyncService) ImportCalibreLibrary(ctx context.Context, calibreDirPath string, libraryID string) (int, error) {
	calibreDirPath, err := resolveCalibreDir(calibreDirPath)
	if err != nil {
		return 0, err
	}
	dbPath, err := localfs.SafeJoin(calibreDirPath, "metadata.db")
	if err != nil {
		return 0, apperrors.New(apperrors.ErrBadRequest, "Invalid Calibre library path")
	}
	books, err := calibre.ReadMetadataDB(ctx, dbPath)
	if err != nil {
		return 0, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}

	importedCount := 0
	for _, cb := range books {
		bookDir, err := localfs.SafeJoin(calibreDirPath, cb.Path)
		if err != nil {
			continue
		}
		entries, err := os.ReadDir(bookDir)
		if err != nil {
			continue
		}
		if s.importCalibreBook(ctx, libraryID, cb, bookDir, entries) {
			importedCount++
		}
	}
	return importedCount, nil
}

func (s *calibreSyncService) importCalibreBook(ctx context.Context, libraryID string, cb calibre.CalibreBookRecord, bookDir string, entries []os.DirEntry) (imported bool) {
	bookID := uuid.Must(uuid.NewV7()).String()
	committed := false
	defer func() {
		if !committed {
			_ = s.bookRepo.DeleteBook(ctx, bookID)
			_ = s.fileRepo.RemoveBookDir(ctx, bookID)
		}
	}()

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return false
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := s.bookRepo.WithTx(tx)

	book := &models.BookEntity{ID: bookID, LibraryID: libraryID, Title: cb.Title, Status: "active"}
	if cb.Description != "" {
		book.Description = &cb.Description
	}
	metadata := map[string]any{
		"calibre_id": cb.ID, "authors": cb.Authors, "publishers": cb.Publishers,
		"languages": cb.Languages, "tags": cb.Tags, "series": cb.Series,
		"series_index": cb.SeriesIndex, "rating": cb.Rating, "isbn": cb.ISBN,
		"lccn": cb.LCCN, "uuid": cb.UUID, "pubdate": cb.PubDate,
		"timestamp": cb.Timestamp, "identifiers": cb.Identifiers,
	}
	if encoded, marshalErr := jsonx.Marshal(metadata); marshalErr == nil {
		value := string(encoded)
		book.MetadataJSON = &value
	} else {
		return false
	}

	if len(cb.Authors) > 0 && strings.TrimSpace(cb.Authors[0]) != "" {
		author, err := txRepo.GetAuthorByName(ctx, cb.Authors[0])
		if err != nil || author == nil {
			author = &models.AuthorEntity{ID: uuid.Must(uuid.NewV7()).String(), Name: cb.Authors[0]}
			if err := txRepo.CreateAuthor(ctx, author); err != nil {
				return false
			}
		}
		book.AuthorID = &author.ID
		book.AuthorName = &author.Name
	}
	if err := txRepo.CreateBook(ctx, book); err != nil {
		return false
	}
	if err := s.syncCalibreRelations(ctx, txRepo, bookID, cb); err != nil {
		return false
	}
	if err := tx.Commit(); err != nil {
		return false
	}
	txRepo.FlushCache(ctx)
	committed = true

	coverPath, err := localfs.SafeJoin(bookDir, "cover.jpg")
	if err != nil {
		return false
	}
	if cover, readErr := os.ReadFile(coverPath); readErr == nil && len(cover) > 0 {
		coverURL, _, saveErr := s.fileRepo.SaveCover(ctx, bookID, ".jpg", cover)
		if saveErr != nil {
			committed = false
			return false
		}
		book.CoverURL = &coverURL
		if err := s.bookRepo.UpdateBook(ctx, book); err != nil {
			committed = false
			return false
		}
	} else if readErr != nil && !os.IsNotExist(readErr) {
		committed = false
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() || !isAllowedBookFormat(strings.ToLower(filepath.Ext(entry.Name()))) {
			continue
		}
		sourcePath, err := localfs.SafeJoin(bookDir, entry.Name())
		if err != nil {
			committed = false
			return false
		}
		src, err := os.Open(sourcePath)
		if err != nil {
			committed = false
			return false
		}
		saved, saveErr := s.fileRepo.SaveBook(ctx, bookID, entry.Name(), src)
		closeErr := src.Close()
		if saveErr != nil || closeErr != nil {
			committed = false
			return false
		}
		hash, _ := s.fileRepo.HashSHA256(ctx, saved.Path)
		state := "managed"
		if err := s.bookRepo.CreateBookFile(ctx, sqlc.CreateBookFileParams{
			ID:        uuid.Must(uuid.NewV7()).String(),
			BookID:    bookID,
			Path:      saved.Path,
			Format:    saved.Format,
			SizeBytes: saved.SizeBytes,
			ModTime:   saved.ModTime,
			Hash:      convert.StrPtrToNullString(&hash),
			State:     convert.StrPtrToNullString(&state),
		}); err != nil {
			committed = false
			return false
		}
	}
	return true
}

func (s *calibreSyncService) syncCalibreRelations(ctx context.Context, repo repositories.BookDBRepository, bookID string, cb calibre.CalibreBookRecord) error {
	if cb.Series != "" {
		series, err := repo.GetSeriesByName(ctx, cb.Series)
		if err != nil || series == nil {
			series = &models.SeriesEntity{ID: uuid.Must(uuid.NewV7()).String(), Name: cb.Series}
			if err := repo.CreateSeries(ctx, series); err != nil {
				return err
			}
		}
		if err := repo.LinkBookSeries(ctx, bookID, series.ID, cb.SeriesIndex); err != nil {
			return err
		}
	}
	for _, name := range cb.Publishers {
		if name == "" {
			continue
		}
		publisher, err := repo.GetPublisherByName(ctx, name)
		if err != nil || publisher == nil {
			publisher = &models.PublisherEntity{ID: uuid.Must(uuid.NewV7()).String(), Name: name}
			if err := repo.CreatePublisher(ctx, publisher); err != nil {
				return err
			}
		}
		if err := repo.LinkBookPublisher(ctx, bookID, publisher.ID); err != nil {
			return err
		}
	}
	for _, name := range cb.Languages {
		if name == "" {
			continue
		}
		language, err := repo.GetLanguageByName(ctx, name)
		if err != nil || language == nil {
			language = &models.LanguageEntity{ID: uuid.Must(uuid.NewV7()).String(), Name: name}
			if err := repo.CreateLanguage(ctx, language); err != nil {
				return err
			}
		}
		if err := repo.LinkBookLanguage(ctx, bookID, language.ID); err != nil {
			return err
		}
	}
	for _, name := range cb.Tags {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tag, err := repo.GetTagByName(ctx, name)
		if err != nil || tag == nil {
			tag = &models.TagEntity{ID: uuid.Must(uuid.NewV7()).String(), Name: name}
			if err := repo.CreateTag(ctx, tag); err != nil {
				return err
			}
		}
		if err := repo.AddBookTag(ctx, bookID, tag.ID); err != nil {
			return err
		}
	}
	return nil
}
