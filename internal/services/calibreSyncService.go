package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/calibre"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
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

func (s *calibreSyncService) ImportCalibreLibrary(ctx context.Context, calibreDirPath string, libraryID string) (int, error) {
	dbPath := filepath.Join(calibreDirPath, "metadata.db")
	books, err := calibre.ReadMetadataDB(ctx, dbPath)
	if err != nil {
		return 0, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}

	importedCount := 0
	for _, cb := range books {
		imported := func() bool {
			newBookID := uuid.Must(uuid.NewV7()).String()

			tx, err := s.txManager.BeginTx(ctx, nil)
			if err != nil {
				return false
			}
			committed := false
			defer func() {
				if !committed {
					_ = tx.Rollback()
				}
			}()

			txRepo := s.bookRepo.WithTx(tx)

			bookEntity := &models.BookEntity{
				ID:        newBookID,
				LibraryID: libraryID,
				Title:     cb.Title,
				Status:    "active",
			}

			if len(cb.Authors) > 0 && cb.Authors[0] != "" {
				primaryAuthor := cb.Authors[0]
				author, err := txRepo.GetAuthorByName(ctx, primaryAuthor)
				if err != nil || author == nil {
					author = &models.AuthorEntity{
						ID:   uuid.Must(uuid.NewV7()).String(),
						Name: primaryAuthor,
					}
					if err := txRepo.CreateAuthor(ctx, author); err == nil {
						bookEntity.AuthorID = &author.ID
						bookEntity.AuthorName = &author.Name
					}
				} else {
					bookEntity.AuthorID = &author.ID
					bookEntity.AuthorName = &author.Name
				}
			}

			if cb.Description != "" {
				desc := cb.Description
				bookEntity.Description = &desc
			}

			metadataMap := map[string]any{
				"calibre_id":   cb.ID,
				"authors":      cb.Authors,
				"publishers":   cb.Publishers,
				"languages":    cb.Languages,
				"tags":         cb.Tags,
				"series":       cb.Series,
				"series_index": cb.SeriesIndex,
				"rating":       cb.Rating,
				"isbn":         cb.ISBN,
				"lccn":         cb.LCCN,
				"uuid":         cb.UUID,
				"pubdate":      cb.PubDate,
				"timestamp":    cb.Timestamp,
				"identifiers":  cb.Identifiers,
			}
			if metaBytes, err := jsonx.Marshal(metadataMap); err == nil {
				metaStr := string(metaBytes)
				bookEntity.MetadataJSON = &metaStr
			}

			if err := txRepo.CreateBook(ctx, bookEntity); err != nil {
				return false
			}

			if cb.Series != "" {
				series, err := txRepo.GetSeriesByName(ctx, cb.Series)
				if err != nil || series == nil {
					series = &models.SeriesEntity{
						ID:   uuid.Must(uuid.NewV7()).String(),
						Name: cb.Series,
					}
					_ = txRepo.CreateSeries(ctx, series)
				}
				_ = txRepo.LinkBookSeries(ctx, newBookID, series.ID, cb.SeriesIndex)
			}

			for _, pubName := range cb.Publishers {
				if pubName == "" {
					continue
				}
				pub, err := txRepo.GetPublisherByName(ctx, pubName)
				if err != nil || pub == nil {
					pub = &models.PublisherEntity{
						ID:   uuid.Must(uuid.NewV7()).String(),
						Name: pubName,
					}
					_ = txRepo.CreatePublisher(ctx, pub)
				}
				_ = txRepo.LinkBookPublisher(ctx, newBookID, pub.ID)
			}

			for _, langCode := range cb.Languages {
				if langCode == "" {
					continue
				}
				lang, err := txRepo.GetLanguageByName(ctx, langCode)
				if err != nil || lang == nil {
					lang = &models.LanguageEntity{
						ID:   uuid.Must(uuid.NewV7()).String(),
						Name: langCode,
					}
					_ = txRepo.CreateLanguage(ctx, lang)
				}
				_ = txRepo.LinkBookLanguage(ctx, newBookID, lang.ID)
			}

			for _, tagName := range cb.Tags {
				if tagName == "" {
					continue
				}
				tag, err := txRepo.GetTagByName(ctx, tagName)
				if err != nil || tag == nil {
					tag = &models.TagEntity{
						ID:   uuid.Must(uuid.NewV7()).String(),
						Name: tagName,
					}
					_ = txRepo.CreateTag(ctx, tag)
				}
				_ = txRepo.AddBookTag(ctx, newBookID, tag.ID)
			}

			bookDir := filepath.Join(calibreDirPath, cb.Path)
			entries, _ := os.ReadDir(bookDir)

			coverPath := filepath.Join(bookDir, "cover.jpg")
			if coverBytes, err := os.ReadFile(coverPath); err == nil && len(coverBytes) > 0 {
				if coverURL, _, err := s.fileRepo.SaveCover(ctx, newBookID, ".jpg", coverBytes); err == nil {
					bookEntity.CoverURL = &coverURL
					_ = txRepo.UpdateBook(ctx, bookEntity)
				}
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				if ext == ".epub" || ext == ".mobi" || ext == ".pdf" || ext == ".docx" || ext == ".cbz" || ext == ".fb2" || ext == ".txt" || ext == ".md" {
					srcFile, err := os.Open(filepath.Join(bookDir, entry.Name()))
					if err != nil {
						continue
					}
					_, _ = s.fileRepo.SaveBook(ctx, newBookID, entry.Name(), srcFile)
					srcFile.Close()
				}
			}

			if err := tx.Commit(); err == nil {
				committed = true
				return true
			}
			return false
		}()

		if imported {
			importedCount++
		}
	}

	return importedCount, nil
}
