package controllers

import (
	"context"
	"time"

	"novelhub/pkg/apperrors"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

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
	audit          services.AuditService
}

func NewBookController(bookService services.BookService, featureService services.FeatureService, settings services.SettingsService, permissions services.PermissionCache, audit services.AuditService) *BookController {
	return &BookController{bookService: bookService, featureService: featureService, settings: settings, permissions: permissions, audit: audit}
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

	res, err := h.bookService.SearchBooksPage(ctx, dto, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(res)
}

func (h *BookController) GetBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	book, err := h.bookService.GetBookWithAccess(ctx, id, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: book.ToResponse()})
}

func (h *BookController) GetBookSeries(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.bookService.GetBookSeriesContext(ctx, c.Params("id"), getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *BookController) DownloadBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	fileDto := &request.BookFileQueryDto{}
	if err := validator.ValidateQueryDto(c, fileDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	filePath, downloadName, err := h.bookService.GetBookFileForDownload(ctx, id, fileDto.FileID, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Download(filePath, downloadName)
}

func (h *BookController) ListBookFiles(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	files, err := h.bookService.ListBookFilesWithAccess(ctx, c.Params("id"), getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: models.BookFileEntitiesToResponse(files)})
}

func (h *BookController) UploadBookFiles(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
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
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Message: "Files uploaded successfully", Data: result.ToResponse()})
}

func (h *BookController) ListChapters(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	chapters, err := h.bookService.ListChaptersWithAccess(ctx, id, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: models.ChapterEntitiesToResponse(chapters)})
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

	results, err := h.bookService.SearchDeep(ctx, dto.Query, int64(dto.Limit), int64(dto.Offset), getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: models.FTSResultEntitiesToResponse(results)})
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

	bookID := c.Params("id")
	searchDto := &request.SearchInBookQueryDto{}
	if err := validator.ValidateQueryDto(c, searchDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	results, err := h.bookService.SearchInBookWithAccess(ctx, bookID, searchDto.Query, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: results})
}

func (h *BookController) SendBookToEmail(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bookID := c.Params("id")
	dto := &request.SendEmailDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}
	if err := h.bookService.SendBookToEmail(ctx, bookID, dto.RecipientEmail, claims); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Email dispatched successfully"})
}

func (h *BookController) EnrichBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bookID := c.Params("id")
	if bookID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Missing book ID"})
	}

	if err := h.bookService.AutoEnrichBook(ctx, bookID); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Book metadata enriched successfully"})
}

func (h *BookController) BatchEnrichBooks(c fiber.Ctx) error {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := h.bookService.BatchEnrichBooks(ctx); err != nil {
			log.Error().Err(err).Msg("failed to run batch book enrichment")
		}
	}()

	return c.JSON(response.CommonResponse{Status: true, Message: "Batch enrichment job started in background"})
}
