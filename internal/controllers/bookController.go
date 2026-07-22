package controllers

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"novelhub/pkg/apperrors"

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

	var cursorTime *time.Time
	if dto.Cursor != "" {
		if t, err := time.Parse(time.RFC3339Nano, dto.Cursor); err == nil {
			cursorTime = &t
		}
	}
	books, err = h.bookService.SearchBooks(ctx, libID, searchStr, dto.Nav, dto.Collection, dto.Chip, dto.Facet, dto.FacetID, cursorTime, int64(dto.Limit))

	if err != nil {
		return apperrors.HandleError(c, err)
	}

	filtered, allowed := h.bookService.FilterReadableBooks(ctx, books, h.claims(c))
	if !allowed {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Login required"})
	}
	var nextCursor *string
	if len(filtered) >= int(dto.Limit) && len(filtered) > 0 {
		c := filtered[len(filtered)-1].CreatedAt.Format(time.RFC3339Nano)
		nextCursor = &c
	}
	return c.JSON(fiber.Map{
		"status":      true,
		"data":        filtered,
		"next_cursor": nextCursor,
	})
}

func (h *BookController) GetBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	book, err := h.bookService.GetBook(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book not found"})
	}
	if !h.bookService.CanReadBook(ctx, book, h.claims(c)) {
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
	if !h.bookService.CanDownloadBook(ctx, book, h.claims(c)) {
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
	return c.Download(file.Path, h.bookService.SafeDownloadFilename(book.Title, ext))
}

func (h *BookController) ListBookFiles(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	book, err := h.bookService.GetBook(ctx, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book not found"})
	}
	if !h.bookService.CanReadBook(ctx, book, h.claims(c)) {
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
	if !h.bookService.CanReadBook(ctx, book, h.claims(c)) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}
	chapters, err := h.bookService.ListChapters(ctx, id)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: chapters})
}


func (h *BookController) claims(c fiber.Ctx) *response.JWTClaims {
	claims, _ := c.Locals("user_claims").(*response.JWTClaims)
	return claims
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
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: results})
}

func (h *BookController) GetDuplicates(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := h.bookService.GetDuplicateGroups(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: results})
}

func (h *BookController) DeleteBookFile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fileID := c.Params("fileID")
	if fileID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Missing file ID"})
	}

	if err := h.bookService.DeleteBookFile(ctx, fileID); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "File deleted successfully"})
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
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Metadata updated successfully"})
}

func (h *BookController) DeleteBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	if err := h.bookService.DeleteBook(ctx, id); err != nil {
		return apperrors.HandleError(c, err)
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
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Message: "Archive state updated"})
}

func (h *BookController) SearchInBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	publicSettings, err := h.settings.Public(ctx)
	if err != nil || publicSettings == nil || !publicSettings.EnableInBookSearch {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{
			Status:  false,
			Message: "in-book search is disabled by system administrator",
		})
	}

	bookID := c.Params("id")
	query := c.Query("q")

	results, err := h.bookService.SearchInBook(ctx, bookID, query)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: results})
}
