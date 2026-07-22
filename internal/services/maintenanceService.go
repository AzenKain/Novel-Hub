package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"


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
}

type maintenanceService struct {
	bookRepo  repositories.BookDBRepository
	fileRepo  repositories.BookFileRepository
	parsers   *bookparser.Registry
	txManager database.TxManager
}

func NewMaintenanceService(bookRepo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, parsers *bookparser.Registry, txManager database.TxManager) MaintenanceService {
	return &maintenanceService{
		bookRepo:  bookRepo,
		fileRepo:  fileRepo,
		parsers:   parsers,
		txManager: txManager,
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
	err := s.bookRepo.DeleteFTSBook(ctx, bookID)
	if err != nil {
		return err
	}

	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil || len(files) == 0 {
		return nil // skip
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

	chapters, err := s.bookRepo.ListChaptersByBook(ctx, bookID)
	if err != nil {
		return err
	}

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

		text := removeHTMLTags(html)

		err = s.bookRepo.InsertFTSChapter(ctx, bookID, ch.ID, ch.Title, text)
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert FTS chapter")
		}
	}

	return nil
}

func (s *maintenanceService) RunMaintenance(ctx context.Context) error {
	limit := int64(100)
	var offset int64 = 0

	for {
		files, err := s.bookRepo.ListAllFiles(ctx, limit, offset)
		if err != nil {
			return err
		}

		if len(files) == 0 {
			break
		}

		for _, f := range files {
			if !s.fileRepo.Exists(ctx, f.Path) {
				log.Info().Str("path", f.Path).Msg("File not found, cleaning up")
				_ = s.bookRepo.DeleteFile(ctx, f.ID)
				count, _ := s.bookRepo.CountFilesForBook(ctx, f.BookID)
				if count == 0 {
					log.Info().Str("book", f.BookID).Msg("Book has no files, cleaning up book folder and record")
					_ = s.fileRepo.RemoveBookDir(ctx, f.BookID)
					_ = s.bookRepo.DeleteFTSBook(ctx, f.BookID)
					_ = s.bookRepo.DeleteBook(ctx, f.BookID)
				}
			}
		}

		offset += limit
		time.Sleep(50 * time.Millisecond) // Yield to avoid CPU/IO starvation
	}

	if err := s.CleanOrphanUploads(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to clean orphan uploads")
	}

	return nil
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

		// If no chunks yet, use directory modtime
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
