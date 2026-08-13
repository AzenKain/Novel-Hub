package controllers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"

	"github.com/gofiber/fiber/v3"
)

type VBookController struct {
	vbookService    services.VBookService
	settingsService services.SettingsService
}

func NewVBookController(vbookService services.VBookService, settingsService services.SettingsService) *VBookController {
	return &VBookController{
		vbookService:    vbookService,
		settingsService: settingsService,
	}
}

func (c *VBookController) GetHome(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sections, err := c.vbookService.GetHomeSections(reqCtx, getBaseURL(ctx, c.settingsService))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   sections,
	})
}

func (c *VBookController) GetGenres(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	genres, err := c.vbookService.GetGenres(reqCtx, getBaseURL(ctx, c.settingsService))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   genres,
	})
}

func (c *VBookController) GetBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var searchPtr *string
	if q := strings.TrimSpace(ctx.Query("search")); q != "" {
		searchPtr = &q
	} else if q := strings.TrimSpace(ctx.Query("q")); q != "" {
		searchPtr = &q
	}

	facet := strings.TrimSpace(ctx.Query("facet"))
	facetID := strings.TrimSpace(ctx.Query("facet_id"))
	sort := strings.TrimSpace(ctx.Query("sort"))

	pageStr := strings.TrimSpace(ctx.Query("page"))

	limit := 20
	if lStr := strings.TrimSpace(ctx.Query("limit")); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	result, err := c.vbookService.GetBooks(reqCtx, getBaseURL(ctx, c.settingsService), searchPtr, sort, facet, facetID, pageStr, limit)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (c *VBookController) SearchBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := strings.TrimSpace(ctx.Query("q"))
	if query == "" {
		query = strings.TrimSpace(ctx.Query("keyword"))
	}

	pageStr := strings.TrimSpace(ctx.Query("page"))

	limit := 20
	if lStr := strings.TrimSpace(ctx.Query("limit")); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	result, err := c.vbookService.SearchBooks(reqCtx, getBaseURL(ctx, c.settingsService), query, pageStr, limit)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (c *VBookController) GetDetail(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := strings.TrimSpace(ctx.Query("id"))
	if bookID == "" {
		bookID = strings.TrimSpace(ctx.Query("book_id"))
	}
	if bookID == "" {
		return apperrors.HandleError(ctx, apperrors.New(apperrors.ErrBadRequest, "Book ID is required"))
	}

	detail, err := c.vbookService.GetBookDetail(reqCtx, getBaseURL(ctx, c.settingsService), bookID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   detail,
	})
}

func (c *VBookController) GetTOC(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := strings.TrimSpace(ctx.Query("id"))
	if bookID == "" {
		bookID = strings.TrimSpace(ctx.Query("book_id"))
	}
	if bookID == "" {
		return apperrors.HandleError(ctx, apperrors.New(apperrors.ErrBadRequest, "Book ID is required"))
	}

	toc, err := c.vbookService.GetTOC(reqCtx, getBaseURL(ctx, c.settingsService), bookID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   toc,
	})
}

func (c *VBookController) GetChapterContent(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := strings.TrimSpace(ctx.Query("book_id"))
	if bookID == "" {
		bookID = strings.TrimSpace(ctx.Query("id"))
	}
	chapterID := strings.TrimSpace(ctx.Query("chapter_id"))

	if bookID == "" || chapterID == "" {
		return apperrors.HandleError(ctx, apperrors.New(apperrors.ErrBadRequest, "book_id and chapter_id are required"))
	}

	content, err := c.vbookService.GetChapterContent(reqCtx, bookID, chapterID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   content,
	})
}

func (c *VBookController) GetPluginJSON(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	plugin, err := c.vbookService.GetPluginJSON(reqCtx, getBaseURL(ctx, c.settingsService))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(plugin)
}

func (c *VBookController) GetPluginZip(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := c.vbookService.GetPluginZip(reqCtx, getBaseURL(ctx, c.settingsService))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	ctx.Set(fiber.HeaderContentType, "application/zip")
	return ctx.Send(data)
}

func (c *VBookController) GetPluginZipAudio(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := c.vbookService.GetPluginZipAudio(reqCtx, getBaseURL(ctx, c.settingsService))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	ctx.Set(fiber.HeaderContentType, "application/zip")
	return ctx.Send(data)
}

func (c *VBookController) GetAudioBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pageStr := strings.TrimSpace(ctx.Query("page"))

	limit := 20
	if lStr := strings.TrimSpace(ctx.Query("limit")); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	result, err := c.vbookService.GetAudioBooks(reqCtx, getBaseURL(ctx, c.settingsService), pageStr, limit)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (c *VBookController) GetAudioPlaylist(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := strings.TrimSpace(ctx.Query("book_id"))
	if bookID == "" {
		bookID = strings.TrimSpace(ctx.Query("id"))
	}
	if bookID == "" {
		return apperrors.HandleError(ctx, apperrors.New(apperrors.ErrBadRequest, "Book ID is required"))
	}

	tracks, err := c.vbookService.GetAudioPlaylist(reqCtx, bookID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   tracks,
	})
}

func (c *VBookController) StreamAudio(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := strings.TrimSpace(ctx.Query("book_id"))
	fileID := strings.TrimSpace(ctx.Query("file_id"))
	if bookID == "" || fileID == "" {
		return apperrors.HandleError(ctx, apperrors.New(apperrors.ErrBadRequest, "book_id and file_id are required"))
	}

	file, err := c.vbookService.ResolveAudioStream(reqCtx, bookID, fileID, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	contentType := rawFileContentType(file.Path)
	ctx.Set("Content-Type", contentType)
	ctx.Set("X-Content-Type-Options", "nosniff")
	ctx.Set("Accept-Ranges", "bytes")
	return ctx.SendFile(file.Path, fiber.SendFile{ByteRange: true})
}
