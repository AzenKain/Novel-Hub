package controllers

import (
	"context"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"novelhub/pkg/apperrors"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/validator"
)

type ReaderController struct {
	bookService services.BookService
	settings    services.SettingsService
	permissions services.PermissionCache
}

func NewReaderController(bookService services.BookService, settings services.SettingsService, permissions services.PermissionCache) *ReaderController {
	return &ReaderController{
		bookService: bookService,
		settings:    settings,
		permissions: permissions,
	}
}

func (h *ReaderController) GetBootstrap(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	fileDto := &request.BookFileQueryDto{}
	if err := validator.ValidateQueryDto(c, fileDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	bootstrap, err := h.bookService.GetReaderBootstrap(ctx, bookID, fileDto.FileID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	if !h.bookService.CanReadBook(ctx, bootstrap.Book, getOptionalClaims(c)) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	return c.JSON(response.CommonResponse{Status: true, Data: bootstrap.ToResponse()})
}

func (h *ReaderController) GetChapter(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	chapterID := decodeRouteParam(c.Params("chapterId"))
	book, err := h.bookService.GetBook(ctx, bookID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	if !h.bookService.CanReadBook(ctx, book, getOptionalClaims(c)) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	fileDto := &request.BookFileQueryDto{}
	if err := validator.ValidateQueryDto(c, fileDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	content, err := h.bookService.GetChapterHTML(ctx, bookID, chapterID, fileDto.FileID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; font-src 'self' data:")
	return c.SendString(content)
}

func (h *ReaderController) GetFile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	book, err := h.bookService.GetBook(ctx, bookID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	if !h.bookService.CanReadBook(ctx, book, getOptionalClaims(c)) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}
	fileDto := &request.BookFileQueryDto{}
	if err := validator.ValidateQueryDto(c, fileDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	file, err := h.bookService.GetBookFile(ctx, bookID, fileDto.FileID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	contentType := rawFileContentType(file.Path)
	c.Set("Content-Type", contentType)
	c.Set("X-Content-Type-Options", "nosniff")
	if strings.HasPrefix(contentType, "text/html") {
		c.Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src data:; style-src 'unsafe-inline'")
	}
	c.Set("Content-Disposition", inlineContentDisposition(filepath.Base(file.Path)))
	c.Set("Accept-Ranges", "bytes")
	return c.SendFile(file.Path, fiber.SendFile{ByteRange: true})
}

func (h *ReaderController) GetAsset(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	assetPath := decodeRouteParam(c.Params("*"))
	book, err := h.bookService.GetBook(ctx, bookID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	if !h.bookService.CanReadBook(ctx, book, getOptionalClaims(c)) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	fileDto := &request.BookFileQueryDto{}
	if err := validator.ValidateQueryDto(c, fileDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	asset, err := h.bookService.GetAsset(ctx, bookID, assetPath, fileDto.FileID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	c.Set("Content-Type", asset.ContentType)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Security-Policy", "default-src 'none'")

	// A comic chapter is one <img> per page, so a 200-page volume means 200 hits
	// here. "private" because access is per-user and a shared proxy must not serve
	// one reader's asset to another; etag.New() handles revalidation after expiry.
	c.Set(fiber.HeaderCacheControl, "private, max-age=3600")
	return c.Send(asset.Data)
}

func decodeRouteParam(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

func (h *ReaderController) ListImages(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	book, err := h.bookService.GetBook(ctx, bookID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	if !h.bookService.CanReadBook(ctx, book, getOptionalClaims(c)) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	fileDto := &request.BookFileQueryDto{}
	if err := validator.ValidateQueryDto(c, fileDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	images, err := h.bookService.ListImages(ctx, bookID, fileDto.FileID)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: images})
}

func (h *ReaderController) UpdateCover(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	bookID := c.Params("id")
	input := request.UpdateCoverDto{
		CoverURL:      c.FormValue("cover_url"),
		EPUBImagePath: c.FormValue("epub_image_path"),
	}
	file, err := c.FormFile("cover")
	if err == nil && file != nil {
		f, err := file.Open()
		if err != nil {
			// The client's part parsed fine; failing to open the spooled copy is our side.
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrInternalError, "Failed to open uploaded file"))
		}
		defer f.Close()
		limit := h.settings.Limits().CoverBytes
		input.UploadedData, err = io.ReadAll(io.LimitReader(f, limit+1))
		if err != nil {
			return apperrors.HandleError(c, err)
		}
		if int64(len(input.UploadedData)) > limit {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.CommonResponse{Status: false, Message: "Cover exceeds size limit"})
		}
		input.UploadedFileName = file.Filename
	}

	coverURLPath, err := h.bookService.UpdateCover(ctx, bookID, input)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Cover updated successfully", Data: fiber.Map{"cover_url": coverURLPath}})
}

func rawFileContentType(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".pdf":
		return "application/pdf"
	case ".txt", ".md", ".markdown":
		return "text/plain; charset=utf-8"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".odt":
		return "application/vnd.oasis.opendocument.text"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".rtf":
		return "application/rtf"
	case ".fb2", ".fbz":
		return "application/x-fictionbook+xml"
	case ".zip":
		return "application/zip"
	case ".cbz":
		return "application/vnd.comicbook+zip"
	case ".cbr":
		return "application/vnd.comicbook-rar"
	case ".cbt":
		return "application/x-tar"
	case ".cb7":
		return "application/x-7z-compressed"
	case ".mobi", ".azw", ".azw3", ".amz":
		return "application/octet-stream"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a", ".m4b":
		return "audio/mp4"
	case ".flac":
		return "audio/flac"
	default:
		if contentType := mime.TypeByExtension(filepath.Ext(filePath)); contentType != "" {
			return contentType
		}
		return "application/octet-stream"
	}
}

func inlineContentDisposition(filename string) string {
	filename = strings.ReplaceAll(filename, `"`, "")
	filename = strings.ReplaceAll(filename, "\r", "")
	filename = strings.ReplaceAll(filename, "\n", "")
	if filename == "" {
		filename = "book"
	}
	return `inline; filename="` + filename + `"`
}
