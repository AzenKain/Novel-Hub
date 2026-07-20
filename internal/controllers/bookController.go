package controllers

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/services"
	"novelhub/pkg/validator"
)

type BookController struct {
	bookService    services.BookService
	featureService services.FeatureService
	settings       services.SettingsService
	permissions    services.PermissionCache
}

func NewBookController(bookService services.BookService, featureService services.FeatureService, settings services.SettingsService, permissions services.PermissionCache) *BookController {
	return &BookController{bookService: bookService, featureService: featureService, settings: settings, permissions: permissions}
}

func (h *BookController) ListBooks(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchBookDto{}
	dto.Limit = 20
	dto.Offset = 0

	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var books []*models.BookEntity
	var err error

	var libID *string
	if dto.LibraryID != "" {
		libID = &dto.LibraryID
	}
	var searchStr *string
	if dto.Search != "" {
		searchStr = &dto.Search
	}

	if dto.Cursor != "" {
		books, err = h.bookService.SearchBooksCursor(ctx, libID, searchStr, dto.Nav, dto.Collection, dto.Chip, dto.Facet, dto.FacetID, dto.Cursor, int64(dto.Limit))
	} else {
		// fallback to calculating offset if page is provided
		offset := dto.Offset
		if offset == 0 && dto.Page > 1 {
			offset = (dto.Page - 1) * dto.Limit
		}
		books, err = h.bookService.SearchBooks(ctx, libID, searchStr, dto.Nav, dto.Collection, dto.Chip, dto.Facet, dto.FacetID, int64(dto.Limit), int64(offset))
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to list books"})
	}

	filtered, allowed := h.filterReadableBooks(c, books)
	if !allowed {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Login required"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: filtered})
}

func (h *BookController) GetBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	book, err := h.bookService.GetBook(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book not found"})
	}
	if !h.canReadBook(c, book) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: book})
}

func (h *BookController) DownloadBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	book, err := h.bookService.GetBook(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book not found"})
	}
	if !h.canDownloadBook(c, book) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Downloads are not allowed"})
	}

	file, err := h.bookService.GetBookFile(ctx, id, c.Query("file_id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book file not found"})
	}
	_ = h.featureService.RecordDownload(ctx, id)

	ext := strings.ToLower(filepath.Ext(file.Path))
	if ext == "" {
		ext = ".epub"
	}
	return c.Download(file.Path, safeDownloadFilename(book.Title, ext))
}

func (h *BookController) ListBookFiles(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	book, err := h.bookService.GetBook(ctx, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book not found"})
	}
	if !h.canReadBook(c, book) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}
	files, err := h.bookService.ListBookFiles(ctx, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book files not found"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: files})
}

func (h *BookController) UploadBookFiles(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Invalid multipart form"})
	}
	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "No files provided"})
	}
	result, err := h.bookService.UploadBookFiles(ctx, c.Params("id"), files)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}
	return c.JSON(response.CommonResponse{Status: true, Message: "Files uploaded successfully", Data: result})
}

func (h *BookController) ListChapters(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	book, err := h.bookService.GetBook(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book not found"})
	}
	if !h.canReadBook(c, book) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}
	chapters, err := h.bookService.ListChapters(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to list chapters"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: chapters})
}

func (h *BookController) filterReadableBooks(c fiber.Ctx, books []*models.BookEntity) ([]*models.BookEntity, bool) {
	if len(books) == 0 {
		return books, true
	}
	if h.claims(c) == nil {
		settings, err := h.settings.Public(c.Context())
		if err == nil && settings.GuestAccess.Mode == "login_required" {
			return nil, false
		}
		out := make([]*models.BookEntity, 0, len(books))
		for _, book := range books {
			if book != nil && h.settings.GuestAllows(book.LibraryID) {
				out = append(out, book)
			}
		}
		return out, true
	}
	out := make([]*models.BookEntity, 0, len(books))
	for _, book := range books {
		if book != nil && h.canReadBook(c, book) {
			out = append(out, book)
		}
	}
	return out, true
}

func (h *BookController) canReadBook(c fiber.Ctx, book *models.BookEntity) bool {
	if book == nil {
		return false
	}
	claims := h.claims(c)
	if claims == nil {
		return h.settings.GuestAllows(book.LibraryID)
	}
	return h.permissions.CanRoles(claims.RoleIDs, claims.Roles, "book.read", map[string]any{"library_id": book.LibraryID})
}

func (h *BookController) canDownloadBook(c fiber.Ctx, book *models.BookEntity) bool {
	if book == nil {
		return false
	}
	claims := h.claims(c)
	if claims == nil {
		return false
	}
	admin := h.permissions.IsAdmin(claims.RoleIDs, claims.Roles)
	if !h.settings.PolicyAllows("download", book.LibraryID, admin) {
		return false
	}
	return h.permissions.CanRoles(claims.RoleIDs, claims.Roles, "book.download", map[string]any{"library_id": book.LibraryID})
}

func (h *BookController) claims(c fiber.Ctx) *response.JWTClaims {
	claims, _ := c.Locals("user_claims").(*response.JWTClaims)
	return claims
}

func safeDownloadFilename(title string, ext string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		name = "book"
	}
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, name)
	name = strings.Trim(name, " .-")
	if name == "" {
		name = "book"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return name + ext
}

func (h *BookController) SearchDeep(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchDeepDto{}
	dto.Limit = 20
	dto.Offset = 0
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	results, err := h.bookService.SearchDeep(ctx, dto.Query, int64(dto.Limit), int64(dto.Offset))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Search failed"})
	}

	return c.JSON(response.CommonResponse{Status: true, Data: results})
}

func (h *BookController) GetDuplicates(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := h.bookService.GetDuplicates(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to fetch duplicates"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: results})
}

func (h *BookController) UpdateMetadata(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	dto := &request.UpdateBookMetadataDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	if err := h.bookService.UpdateMetadata(ctx, id, dto); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to update metadata"})
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Metadata updated successfully"})
}

func (h *BookController) DeleteBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	if err := h.bookService.DeleteBook(ctx, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to delete book"})
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Book deleted successfully"})
}

func (h *BookController) ArchiveBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	dto := &request.ArchiveBookDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	if err := h.bookService.ArchiveBook(ctx, id, dto.Archived); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to update archive state"})
	}
	return c.JSON(response.CommonResponse{Status: true, Message: "Archive state updated"})
}
