package services

import (
	"context"
	"os"
	"path/filepath"

	"novelhub/pkg/apperrors"
	"novelhub/pkg/config"
	"novelhub/pkg/localfs"
)

func (s *libraryService) SetupInbox(ctx context.Context, id string) (string, error) {
	lib, err := s.libraryRepo.GetLibrary(ctx, id)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return "", apperrors.New(apperrors.ErrNotFound, "Library not found")
		}
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to load library")
	}

	inboxRoot, err := localfs.SafeJoin(config.GetConfigWithDefault("DATA_DIR", "./data"), "inbox")
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to resolve inbox path")
	}

	libraryPath, err := localfs.SafeJoin(inboxRoot, lib.ID)
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to resolve library inbox path")
	}

	if err := os.MkdirAll(libraryPath, 0750); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to create inbox directory")
	}

	absPath, err := filepath.Abs(libraryPath)
	if err != nil {
		return libraryPath, nil
	}
	return absPath, nil
}
