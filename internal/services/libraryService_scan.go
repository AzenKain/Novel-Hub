package services

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"novelhub/pkg/config"
	"novelhub/pkg/localfs"
)

const inboxSettleDelay = 10 * time.Second

const inboxMaxDepth = 5

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
	imported := 0
	for _, filePath := range collectInboxFiles(libraryPath, 0, s.parsers.HasPath) {
		if err := ctx.Err(); err != nil {
			return imported
		}
		filename := filepath.Base(filePath)
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
	pruneEmptyInboxDirs(libraryPath, 0)
	return imported
}

func collectInboxFiles(dirPath string, depth int, isParsable func(string) bool) []string {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		log.Error().Err(err).Str("dir", dirPath).Msg("failed to read inbox folder")
		return nil
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		// A symlink can point outside the inbox; SafeJoin only guards the path string.
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		entryPath, err := localfs.SafeJoin(dirPath, name)
		if err != nil {
			continue
		}

		if entry.IsDir() {
			if depth >= inboxMaxDepth {
				log.Warn().Str("dir", entryPath).Int("max_depth", inboxMaxDepth).Msg("inbox folder nested too deep, skipping")
				continue
			}
			files = append(files, collectInboxFiles(entryPath, depth+1, isParsable)...)
			continue
		}

		if !isParsable(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) < inboxSettleDelay {
			continue
		}
		files = append(files, entryPath)
	}

	return files
}

func pruneEmptyInboxDirs(dirPath string, depth int) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		childPath, err := localfs.SafeJoin(dirPath, entry.Name())
		if err != nil {
			continue
		}
		pruneEmptyInboxDirs(childPath, depth+1)
	}
	if depth == 0 {
		return
	}
	_ = os.Remove(dirPath)
}
