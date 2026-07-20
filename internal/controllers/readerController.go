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

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/services"
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

// GetBootstrap returns the basic info needed to start the reader (TOC, Book info)
func (h *ReaderController) GetBootstrap(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	bootstrap, err := h.bookService.GetReaderBootstrap(ctx, bookID, c.Query("file_id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Book not found"})
	}
	if !h.canRead(c, bootstrap.Book) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	return c.JSON(response.CommonResponse{Status: true, Data: bootstrap})
}

// GetChapter returns the HTML content for a specific chapter, with rewritten asset links
func (h *ReaderController) GetChapter(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	chapterID := decodeRouteParam(c.Params("chapterId"))
	if !h.canReadBookID(ctx, c, bookID) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	content, err := h.bookService.GetChapterHTML(ctx, bookID, chapterID, c.Query("file_id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to read chapter content"})
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(content)
}

// GetFile streams the selected source file inline for browser-native readers such as PDF.
func (h *ReaderController) GetFile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	if !h.canReadBookID(ctx, c, bookID) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}
	file, err := h.bookService.GetBookFile(ctx, bookID, c.Query("file_id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "File not found"})
	}

	c.Set("Content-Type", rawFileContentType(file.Path))
	c.Set("Content-Disposition", inlineContentDisposition(filepath.Base(file.Path)))
	c.Set("Accept-Ranges", "bytes")
	return c.SendFile(file.Path, fiber.SendFile{ByteRange: true})
}

// GetAsset returns a raw asset (image, css) from the EPUB
func (h *ReaderController) GetAsset(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	assetPath := decodeRouteParam(c.Params("*")) // Fiber wildcard for the rest of the path
	if !h.canReadBookID(ctx, c, bookID) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	asset, err := h.bookService.GetAsset(ctx, bookID, assetPath, c.Query("file_id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Asset not found"})
	}

	c.Set("Content-Type", asset.ContentType)
	return c.Send(asset.Data)
}

func decodeRouteParam(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

// ListImages returns a list of image paths inside the EPUB
func (h *ReaderController) ListImages(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	if !h.canReadBookID(ctx, c, bookID) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	images, err := h.bookService.ListImages(ctx, bookID, c.Query("file_id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to list images"})
	}

	return c.JSON(response.CommonResponse{Status: true, Data: images})
}

func (h *ReaderController) canReadBookID(ctx context.Context, c fiber.Ctx, bookID string) bool {
	book, err := h.bookService.GetBook(ctx, bookID)
	if err != nil {
		return false
	}
	return h.canRead(c, book)
}

func (h *ReaderController) canRead(c fiber.Ctx, book *models.BookEntity) bool {
	if book == nil {
		return false
	}
	claims, _ := c.Locals("user_claims").(*response.JWTClaims)
	if claims == nil {
		return h.settings.GuestAllows(book.LibraryID)
	}
	return h.permissions.CanRoles(claims.RoleIDs, claims.Roles, "book.read", map[string]any{"library_id": book.LibraryID})
}

// UpdateCover accepts a cover image upload or URL and saves it
func (h *ReaderController) UpdateCover(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	input := services.UpdateCoverInput{
		CoverURL:      c.FormValue("cover_url"),
		EPUBImagePath: c.FormValue("epub_image_path"),
	}
	file, err := c.FormFile("cover")
	if err == nil && file != nil {
		f, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Failed to open uploaded file"})
		}
		defer f.Close()
		input.UploadedData, err = io.ReadAll(f)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to read uploaded file"})
		}
		input.UploadedFileName = file.Filename
	}

	coverURLPath, err := h.bookService.UpdateCover(ctx, bookID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
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
