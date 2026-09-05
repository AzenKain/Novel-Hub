package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/webdav"
)

type WebDAVService interface {
	ResolvePath(ctx context.Context, relativePath string, claims *response.JWTClaims, depth int) ([]webdav.WebDAVNode, error)
	GetBookFile(ctx context.Context, relativePath string, claims *response.JWTClaims) (filePath string, mimeType string, downloadName string, err error)
}

type webdavService struct {
	libraryService LibraryService
	bookService    BookService
	permissions    PermissionCache
	settings       SettingsService
}

func NewWebDAVService(
	libraryService LibraryService,
	bookService BookService,
	permissions PermissionCache,
	settings SettingsService,
) WebDAVService {
	return &webdavService{
		libraryService: libraryService,
		bookService:    bookService,
		permissions:    permissions,
		settings:       settings,
	}
}

func (s *webdavService) canAccessWebDAV(ctx context.Context, claims *response.JWTClaims) bool {
	c := resolveClaims(claims)
	if isGuestClaims(c) && s.settings != nil {
		settings, err := s.settings.Public(ctx)
		if err == nil && settings != nil && settings.GuestLoginRequired {
			return false
		}
	}
	if s.permissions != nil {
		if s.permissions.IsAdmin(c.RoleIDs, c.Roles) {
			return true
		}
		return s.permissions.CanRoles(c.RoleIDs, c.Roles, constants.PermWebDAVRead, nil)
	}
	return true
}

func (s *webdavService) canDownloadWebDAV(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	if book == nil {
		return false
	}
	c := resolveClaims(claims)
	if s.permissions != nil {
		if s.permissions.IsAdmin(c.RoleIDs, c.Roles) {
			return true
		}
		if !s.bookService.CanDownloadBook(ctx, book, claims) {
			return false
		}
		return s.permissions.CanRoles(c.RoleIDs, c.Roles, constants.PermWebDAVDownload, map[string]any{"library_id": book.LibraryID})
	}
	return true
}

func (s *webdavService) ResolvePath(ctx context.Context, relativePath string, claims *response.JWTClaims, depth int) ([]webdav.WebDAVNode, error) {
	if !s.canAccessWebDAV(ctx, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Forbidden to access WebDAV")
	}

	cleanPath := webdav.SanitizeWebDAVPath(relativePath)

	basePrefix := "/webdav"
	if strings.HasPrefix(relativePath, "/api/webdav") {
		basePrefix = "/api/webdav"
	}

	if cleanPath == "/" || cleanPath == "" {
		rootNode := webdav.WebDAVNode{
			Href:        basePrefix + "/",
			DisplayName: "NovelHub",
			IsDir:       true,
			ModTime:     time.Now().UTC(),
		}

		if depth == 0 {
			return []webdav.WebDAVNode{rootNode}, nil
		}

		libraries, err := s.libraryService.ListLibraries(ctx, claims)
		if err != nil {
			return nil, err
		}

		nodes := make([]webdav.WebDAVNode, 0, len(libraries)+1)
		nodes = append(nodes, rootNode)

		for _, lib := range libraries {
			libHref := fmt.Sprintf("%s/%s/", basePrefix, lib.Name)
			nodes = append(nodes, webdav.WebDAVNode{
				Href:        libHref,
				DisplayName: lib.Name,
				IsDir:       true,
				ModTime:     lib.UpdatedAt,
			})
		}

		return nodes, nil
	}

	parts := strings.Split(strings.Trim(cleanPath, "/"), "/")
	libraryName := parts[0]

	libraries, err := s.libraryService.ListLibraries(ctx, claims)
	if err != nil {
		return nil, err
	}

	var targetLib *response.LibraryResponse
	for _, lib := range libraries {
		if strings.EqualFold(lib.Name, libraryName) || lib.ID == libraryName {
			targetLib = lib
			break
		}
	}

	if targetLib == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Library not found")
	}

	if len(parts) == 1 {
		libHref := fmt.Sprintf("%s/%s/", basePrefix, targetLib.Name)
		libraryNode := webdav.WebDAVNode{
			Href:        libHref,
			DisplayName: targetLib.Name,
			IsDir:       true,
			ModTime:     targetLib.UpdatedAt,
		}

		if depth == 0 {
			return []webdav.WebDAVNode{libraryNode}, nil
		}

		userID := ""
		if claims != nil {
			userID = claims.UId
		}
		books, err := s.bookService.SearchBooks(ctx, &targetLib.ID, nil, "", "", "", "", "", "title_asc", "", 100, userID)
		if err != nil {
			return nil, err
		}

		nodes := make([]webdav.WebDAVNode, 0, len(books)+1)
		nodes = append(nodes, libraryNode)

		for _, book := range books {
			if !s.bookService.CanReadBook(ctx, book, claims) {
				continue
			}

			files, err := s.bookService.ListBookFiles(ctx, book.ID)
			if err != nil || len(files) == 0 {
				continue
			}

			for _, file := range files {
				ext := filepath.Ext(file.Path)
				if ext == "" {
					ext = "." + file.Format
				}
				fileName := s.bookService.SafeDownloadFilename(book.Title, ext)
				fileHref := fmt.Sprintf("%s/%s/%s", basePrefix, targetLib.Name, fileName)

				modTime := file.ModTime
				if modTime.IsZero() {
					modTime = book.UpdatedAt
				}

				nodes = append(nodes, webdav.WebDAVNode{
					Href:        fileHref,
					DisplayName: fileName,
					IsDir:       false,
					Size:        file.SizeBytes,
					ContentType: guessWebDAVMimeType(file.Path, file.Format),
					ModTime:     modTime,
					ETag:        fmt.Sprintf("%s-%d", file.ID, modTime.Unix()),
				})
			}
		}

		return nodes, nil
	}

	fileName := parts[1]
	userID := ""
	if claims != nil {
		userID = claims.UId
	}
	books, err := s.bookService.SearchBooks(ctx, &targetLib.ID, nil, "", "", "", "", "", "title_asc", "", 100, userID)
	if err != nil {
		return nil, err
	}

	for _, book := range books {
		if !s.bookService.CanReadBook(ctx, book, claims) {
			continue
		}

		files, err := s.bookService.ListBookFiles(ctx, book.ID)
		if err != nil {
			continue
		}

		for _, file := range files {
			ext := filepath.Ext(file.Path)
			if ext == "" {
				ext = "." + file.Format
			}
			expectedName := s.bookService.SafeDownloadFilename(book.Title, ext)

			if strings.EqualFold(expectedName, fileName) || strings.EqualFold(file.ID, fileName) {
				fileHref := fmt.Sprintf("%s/%s/%s", basePrefix, targetLib.Name, expectedName)
				modTime := file.ModTime
				if modTime.IsZero() {
					modTime = book.UpdatedAt
				}

				return []webdav.WebDAVNode{
					{
						Href:        fileHref,
						DisplayName: expectedName,
						IsDir:       false,
						Size:        file.SizeBytes,
						ContentType: guessWebDAVMimeType(file.Path, file.Format),
						ModTime:     modTime,
						ETag:        fmt.Sprintf("%s-%d", file.ID, modTime.Unix()),
					},
				}, nil
			}
		}
	}

	return nil, apperrors.New(apperrors.ErrNotFound, "File not found")
}

func (s *webdavService) GetBookFile(ctx context.Context, relativePath string, claims *response.JWTClaims) (filePath string, mimeType string, downloadName string, err error) {
	if !s.canAccessWebDAV(ctx, claims) {
		return "", "", "", apperrors.New(apperrors.ErrForbidden, "Forbidden to access WebDAV")
	}

	cleanPath := webdav.SanitizeWebDAVPath(relativePath)
	parts := strings.Split(strings.Trim(cleanPath, "/"), "/")
	if len(parts) < 2 {
		return "", "", "", apperrors.New(apperrors.ErrBadRequest, "Invalid file path")
	}

	libraryName := parts[0]
	fileName := parts[1]

	libraries, err := s.libraryService.ListLibraries(ctx, claims)
	if err != nil {
		return "", "", "", err
	}

	var targetLib *response.LibraryResponse
	for _, lib := range libraries {
		if strings.EqualFold(lib.Name, libraryName) || lib.ID == libraryName {
			targetLib = lib
			break
		}
	}

	if targetLib == nil {
		return "", "", "", apperrors.New(apperrors.ErrNotFound, "Library not found")
	}

	userID := ""
	if claims != nil {
		userID = claims.UId
	}
	books, err := s.bookService.SearchBooks(ctx, &targetLib.ID, nil, "", "", "", "", "", "title_asc", "", 100, userID)
	if err != nil {
		return "", "", "", err
	}

	for _, book := range books {
		if !s.bookService.CanReadBook(ctx, book, claims) || !s.canDownloadWebDAV(ctx, book, claims) {
			continue
		}

		files, err := s.bookService.ListBookFiles(ctx, book.ID)
		if err != nil {
			continue
		}

		for _, file := range files {
			ext := filepath.Ext(file.Path)
			if ext == "" {
				ext = "." + file.Format
			}
			expectedName := s.bookService.SafeDownloadFilename(book.Title, ext)

			if strings.EqualFold(expectedName, fileName) || strings.EqualFold(file.ID, fileName) {
				return file.Path, guessWebDAVMimeType(file.Path, file.Format), expectedName, nil
			}
		}
	}

	return "", "", "", apperrors.New(apperrors.ErrNotFound, "Book file not found")
}

func guessWebDAVMimeType(filePath, format string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		ext = "." + strings.ToLower(format)
	}

	switch ext {
	case ".epub":
		return "application/epub+zip"
	case ".pdf":
		return "application/pdf"
	case ".mobi":
		return "application/x-mobipocket-ebook"
	case ".azw", ".azw3":
		return "application/vnd.amazon.ebook"
	case ".cbz":
		return "application/vnd.comicbook+zip"
	case ".cbr":
		return "application/vnd.comicbook-rar"
	case ".fb2":
		return "application/x-fb2+xml"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".mp3":
		return "audio/mpeg"
	case ".m4b", ".m4a":
		return "audio/mp4"
	case ".flac":
		return "audio/flac"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}
