package services

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"

	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
)

type MaintenanceService interface {
	HashFile(ctx context.Context, fileID string) error
	IndexBook(ctx context.Context, bookID string) error
	RunMaintenance(ctx context.Context) error
}

type maintenanceService struct {
	bookRepo repositories.BookDBRepository
	fileRepo repositories.BookFileRepository
	parsers  *bookparser.Registry
}

func NewMaintenanceService(bookRepo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, parsers *bookparser.Registry) MaintenanceService {
	return &maintenanceService{
		bookRepo: bookRepo,
		fileRepo: fileRepo,
		parsers:  parsers,
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
	files, err := s.bookRepo.ListAllFiles(ctx)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !s.fileRepo.Exists(ctx, f.Path) {
			log.Info().Str("path", f.Path).Msg("File not found, cleaning up")
			_ = s.bookRepo.DeleteFile(ctx, f.ID)

			// Check if book has other files
			count, _ := s.bookRepo.CountFilesForBook(ctx, f.BookID)
			if count == 0 {
				log.Info().Str("book", f.BookID).Msg("Book has no files, cleaning up book")
				_ = s.bookRepo.DeleteBook(ctx, f.BookID)
			}
		}
	}

	return nil
}
