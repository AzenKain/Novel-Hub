package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"

	"novelhub/pkg/config"

	"github.com/google/uuid"
)

type UploadService interface {
	InitUploadSession(ctx context.Context) (string, error)
	SaveChunk(ctx context.Context, uploadID string, chunkIndexStr string, file *multipart.FileHeader) error
	CommitUpload(ctx context.Context, uploadID, target, libraryID, bookID, filename string, totalChunks int) error
}

type uploadService struct {
	libraryService LibraryService
	bookService    BookService
}

func NewUploadService(libraryService LibraryService, bookService BookService) UploadService {
	return &uploadService{
		libraryService: libraryService,
		bookService:    bookService,
	}
}

func (s *uploadService) InitUploadSession(ctx context.Context) (string, error) {
	uploadID := uuid.Must(uuid.NewV7()).String()

	uploadDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads", uploadID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to initialize upload: %w", err)
	}

	return uploadID, nil
}

func (s *uploadService) SaveChunk(ctx context.Context, uploadID string, chunkIndexStr string, file *multipart.FileHeader) error {
	if _, err := uuid.Parse(uploadID); err != nil {
		return fmt.Errorf("invalid upload session ID")
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		return fmt.Errorf("invalid chunk index")
	}

	uploadDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads", uploadID)
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		return fmt.Errorf("upload session not found or expired")
	}

	chunkPath := filepath.Join(uploadDir, fmt.Sprintf("chunk_%d", chunkIndex))

	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open chunk file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(chunkPath)
	if err != nil {
		return fmt.Errorf("failed to create chunk file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to save chunk: %w", err)
	}

	return nil
}

func (s *uploadService) CommitUpload(ctx context.Context, uploadID, target, libraryID, bookID, filename string, totalChunks int) error {
	if _, err := uuid.Parse(uploadID); err != nil {
		return fmt.Errorf("invalid upload session ID")
	}

	cleanFilename := filepath.Base(filename)
	if cleanFilename == "." || cleanFilename == "/" || cleanFilename == "" {
		return fmt.Errorf("invalid filename")
	}

	uploadDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads", uploadID)
	finalPath := filepath.Join(uploadDir, "merged_file")

	// Clean up unconditionally
	defer os.RemoveAll(uploadDir)

	// Merge chunks
	out, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("failed to create merged file")
	}

	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(uploadDir, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			out.Close()
			return fmt.Errorf("missing chunk %d", i)
		}
		if _, err := io.Copy(out, chunkFile); err != nil {
			out.Close()
			chunkFile.Close()
			return fmt.Errorf("failed to merge chunks")
		}
		chunkFile.Close()
	}
	out.Close()

	switch target {
	case "library":
		if libraryID == "" {
			return fmt.Errorf("missing library ID")
		}
		if err := s.libraryService.ProcessSingleLocalFile(ctx, libraryID, cleanFilename, finalPath); err != nil {
			return fmt.Errorf("failed to process file for library: %w", err)
		}
	case "book":
		if bookID == "" {
			return fmt.Errorf("missing book ID")
		}
		if err := s.bookService.ProcessSingleLocalFile(ctx, bookID, cleanFilename, finalPath); err != nil {
			return fmt.Errorf("failed to process file for book: %w", err)
		}
	default:
		return fmt.Errorf("invalid target")
	}

	return nil
}
