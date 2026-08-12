package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/config"
	"novelhub/pkg/database"
)

type MaintenanceService interface {
	HashFile(ctx context.Context, fileID string) error
	IndexBook(ctx context.Context, bookID string) error
	RunMaintenance(ctx context.Context) error
	CleanOrphanUploads(ctx context.Context) error
	CleanEmptyBookDirs(ctx context.Context) error
	CheckDatabaseHealth(ctx context.Context) error
}

type maintenanceService struct {
	bookRepo      repositories.BookDBRepository
	fileRepo      repositories.BookFileRepository
	magicCodeRepo repositories.MagicCodeRepository
	parsers       *bookparser.Registry
	txManager     database.TxManager
}

func NewMaintenanceService(bookRepo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, magicCodeRepo repositories.MagicCodeRepository, parsers *bookparser.Registry, txManager database.TxManager) MaintenanceService {
	return &maintenanceService{
		bookRepo:      bookRepo,
		fileRepo:      fileRepo,
		magicCodeRepo: magicCodeRepo,
		parsers:       parsers,
		txManager:     txManager,
	}
}

func (s *maintenanceService) HashFile(ctx context.Context, fileID string) error {
	file, err := s.bookRepo.GetBookFileById(ctx, fileID)
	if err != nil {
		return err
	}

	hashStr, err := s.fileRepo.HashSHA256(ctx, file.Path)
	if err != nil {
		return err
	}

	return s.bookRepo.UpdateBookFileHash(ctx, fileID, hashStr)
}

func removeHTMLTags(input string) string {
	var builder strings.Builder
	inTag := false
	for _, r := range input {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func (s *maintenanceService) IndexBook(ctx context.Context, bookID string) error {
	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil || len(files) == 0 {
		return nil
	}
	var parser bookparser.Parser
	var filePath string
	for _, file := range files {
		if candidate, ok := s.parsers.ParserForFormat(file.Format); ok && strings.EqualFold(file.Format, "epub") {
			parser = candidate
			filePath = file.Path
			break
		}
	}
	if filePath == "" {
		for _, file := range files {
			candidate, err := s.parsers.Parser(file.Format, file.Path)
			if err != nil {
				continue
			}
			parser = candidate
			filePath = file.Path
			break
		}
		if filePath == "" {
			return nil
		}
	}

	if spine, err := parser.ParseSpine(filePath); err == nil && len(spine) > 0 {
		existingChapters, listErr := s.bookRepo.ListChaptersByBook(ctx, bookID)
		needsUpdate := listErr != nil || len(existingChapters) != len(spine)
		if !needsUpdate {
			for i, ch := range spine {
				if i >= len(existingChapters) || existingChapters[i].Title != ch.Title {
					needsUpdate = true
					break
				}
			}
		}

		if needsUpdate {
			chTx, txErr := s.txManager.BeginTx(ctx, nil)
			if txErr == nil {
				txRepo := s.bookRepo.WithTx(chTx)
				if delErr := txRepo.DeleteChaptersByBook(ctx, bookID); delErr == nil {
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
					if commitErr := chTx.Commit(); commitErr == nil {
						txRepo.FlushCache(ctx)
					} else {
						_ = chTx.Rollback()
					}
				} else {
					_ = chTx.Rollback()
				}
			}
		}
	}

	chapters, err := s.bookRepo.ListChaptersByBook(ctx, bookID)
	if err != nil {
		return err
	}

	type chapterText struct {
		id    string
		title string
		text  string
	}
	texts := make([]chapterText, 0, len(chapters))

	for _, ch := range chapters {
		if ch.ContentPath == nil {
			continue
		}
		if *ch.ContentPath == bookparser.RawFileContentPath {
			continue
		}

		html, err := parser.GetChapterContent(filePath, *ch.ContentPath)
		if err != nil {
			log.Warn().Err(err).Str("book", bookID).Str("ch", ch.Title).Msg("Failed to read chapter for index")
			continue
		}

		texts = append(texts, chapterText{
			id:    ch.ID,
			title: ch.Title,
			text:  removeHTMLTags(html),
		})
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	txRepo := s.bookRepo.WithTx(tx)

	if err := txRepo.DeleteFTSBook(ctx, bookID); err != nil {
		return err
	}

	for _, t := range texts {
		if err := txRepo.InsertFTSChapter(ctx, bookID, t.id, t.title, t.text); err != nil {
			log.Error().Err(err).Str("book", bookID).Str("ch", t.title).Msg("Failed to insert FTS chapter")
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	txRepo.FlushCache(ctx)
	return nil
}

func (s *maintenanceService) RunMaintenance(ctx context.Context) error {
	const limit int64 = 100
	var offset int64

	for {
		files, err := s.bookRepo.ListAllFiles(ctx, limit, offset)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			break
		}

		var deleted int64
		for _, file := range files {
			if !s.fileRepo.Exists(ctx, file.Path) {
				log.Info().Str("path", file.Path).Msg("File not found, cleaning up")
				bookDeleted, err := s.removeMissingFile(ctx, file.ID, file.BookID)
				if err != nil {
					return err
				}
				deleted++
				if bookDeleted {
					if err := s.fileRepo.RemoveBookDir(ctx, file.BookID); err != nil {
						log.Warn().Err(err).Str("book_id", file.BookID).Msg("failed to remove orphaned book directory")
					}
				}
			} else {
				if err := s.IndexBook(ctx, file.BookID); err != nil {
					log.Warn().Err(err).Str("book_id", file.BookID).Msg("failed to re-index book during maintenance")
				}
			}
		}

		offset += int64(len(files)) - deleted
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	if err := s.CleanOrphanUploads(ctx); err != nil {
		return err
	}
	if err := s.CleanEmptyBookDirs(ctx); err != nil {
		return err
	}
	if err := s.magicCodeRepo.DeleteExpired(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to delete expired magic codes")
	}

	return nil
}

func (s *maintenanceService) removeMissingFile(ctx context.Context, fileID, bookID string) (bool, error) {
	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	repo := s.bookRepo.WithTx(tx)
	if err := repo.DeleteFile(ctx, fileID); err != nil {
		return false, err
	}
	count, err := repo.CountFilesForBook(ctx, bookID)
	if err != nil {
		return false, err
	}
	bookDeleted := count == 0
	if bookDeleted {
		if err := repo.DeleteFTSBook(ctx, bookID); err != nil {
			return false, err
		}
		if err := repo.DeleteBook(ctx, bookID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	repo.FlushCache(ctx)
	return bookDeleted, nil
}

func (s *maintenanceService) CleanEmptyBookDirs(ctx context.Context) error {
	removed, err := s.fileRepo.RemoveEmptyBookDirs(ctx)
	if err != nil {
		return err
	}
	if removed > 0 {
		log.Info().Int("removed", removed).Msg("Removed empty book directories")
	}
	return nil
}

func (s *maintenanceService) CheckDatabaseHealth(ctx context.Context) error {
	_, err := sqlc.New(s.txManager.DB()).DatabaseHealthCheck(ctx)
	return err
}

func (s *maintenanceService) CleanOrphanUploads(ctx context.Context) error {
	uploadDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads")
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-2 * time.Hour)
	count := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		targetPath := filepath.Join(uploadDir, entry.Name())
		chunks, err := os.ReadDir(targetPath)
		if err != nil {
			continue
		}

		var newest time.Time
		for _, chunk := range chunks {
			info, err := chunk.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(newest) {
				newest = info.ModTime()
			}
		}

		if newest.IsZero() {
			info, err := entry.Info()
			if err == nil {
				newest = info.ModTime()
			}
		}

		if newest.Before(cutoff) {
			if err := os.RemoveAll(targetPath); err != nil {
				log.Error().Err(err).Str("path", targetPath).Msg("Failed to clean orphan upload dir")
			} else {
				count++
			}
		}
	}

	if count > 0 {
		log.Info().Int("cleaned", count).Msg("Cleaned orphan upload directories")
	}

	return nil
}
