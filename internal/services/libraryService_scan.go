package services

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"novelhub/pkg/config"
	"novelhub/pkg/localfs"
)

const inboxSettleDelay = 10 * time.Second

func (s *libraryService) ScanInbox(ctx context.Context) (int, error) {
	inboxRoot, err := localfs.SafeJoin(config.GetConfigWithDefault("DATA_DIR", "./data"), "inbox")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(inboxRoot, 0750); err != nil {
		return 0, err
	}

	libraryDirs, err := os.ReadDir(inboxRoot)
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, libraryDir := range libraryDirs {
		if err := ctx.Err(); err != nil {
			return imported, err
		}
		if !libraryDir.IsDir() {
			continue
		}
		libraryID := libraryDir.Name()
		if _, err := s.libraryRepo.GetLibrary(ctx, libraryID); err != nil {
			log.Warn().Str("dir", libraryID).Msg("inbox folder does not match any library, skipping")
			continue
		}
		libraryPath, err := localfs.SafeJoin(inboxRoot, libraryID)
		if err != nil {
			continue
		}
		imported += s.scanInboxLibrary(ctx, libraryID, libraryPath)
	}

	return imported, nil
}

func (s *libraryService) scanInboxLibrary(ctx context.Context, libraryID string, libraryPath string) int {
	entries, err := os.ReadDir(libraryPath)
	if err != nil {
		log.Error().Err(err).Str("library_id", libraryID).Msg("failed to read inbox folder")
		return 0
	}

	imported := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return imported
		}
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !s.parsers.HasPath(filename) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) < inboxSettleDelay {
			continue
		}
		filePath, err := localfs.SafeJoin(libraryPath, filename)
		if err != nil {
			continue
		}

		if err := s.ProcessSingleLocalFile(ctx, libraryID, filename, filePath); err != nil {
			log.Error().Err(err).Str("library_id", libraryID).Str("file", filename).Msg("failed to import inbox file")
			continue
		}
		if err := os.Remove(filePath); err != nil {
			log.Warn().Err(err).Str("file", filePath).Msg("imported inbox file could not be removed; it may be imported again")
		}
		imported++
		log.Info().Str("library_id", libraryID).Str("file", filename).Msg("imported file from inbox")
	}

	return imported
}
