package controllers

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"

	"github.com/gofiber/fiber/v3"
)

type OPDSController struct {
	opdsService     services.OPDSService
	settingsService services.SettingsService
}

func opdsPageQuery(ctx fiber.Ctx) request.OPDSPageDto {
	limit := int64(constants.OPDSDefaultPageSize)
	if raw := strings.TrimSpace(ctx.Query("limit")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > constants.MaxPaginationLimit {
		limit = constants.MaxPaginationLimit
	}
	return request.OPDSPageDto{Cursor: strings.TrimSpace(ctx.Query("cursor")), Limit: limit}
}

func unescapeParam(raw string) string {
	decoded, err := url.QueryUnescape(raw)
	if err == nil && decoded != "" {
		return strings.TrimSpace(decoded)
	}
	if pathDecoded, err := url.PathUnescape(raw); err == nil && pathDecoded != "" {
		return strings.TrimSpace(pathDecoded)
	}
	return strings.TrimSpace(raw)
}

func GetOPDSBasePath(ctx fiber.Ctx) string {
	path := ctx.Path()
	if strings.HasPrefix(path, "/api/opds") {
		return "/api/opds"
	}
	return "/opds"
}

func getOPDSBasePath(ctx fiber.Ctx) string {
	return GetOPDSBasePath(ctx)
}

func NewOPDSController(opdsService services.OPDSService, settingsService services.SettingsService) *OPDSController {
	return &OPDSController{
		opdsService:     opdsService,
		settingsService: settingsService,
	}
}

func (c *OPDSController) GetRootCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetRootCatalog(reqCtx, serverURL, basePath, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetAllBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetAllBooks(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS books catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetRecentBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetRecentBooks(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetHotBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetHotBooks(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate hot books OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetRandomBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetRandomBooks(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate random books OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetOpenSearchDescription(ctx fiber.Ctx) error {
	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	desc := c.opdsService.GetOpenSearchDescription(serverURL, basePath)
	xmlBytes, err := xml.MarshalIndent(desc, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal OpenSearch XML")
	}
	ctx.Set(fiber.HeaderContentType, "application/opensearchdescription+xml; charset=utf-8")
	return ctx.Send(append([]byte(xml.Header), xmlBytes...))
}

func (c *OPDSController) SearchCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := ctx.Query("q", "")
	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.SearchBooksOPDS(reqCtx, serverURL, basePath, query, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to execute OPDS search")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetAuthorsCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetAuthorsCatalog(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate authors OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetAuthorBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authorName := unescapeParam(ctx.Params("name"))
	if queryVal := strings.TrimSpace(ctx.Query("name")); queryVal != "" {
		authorName = queryVal
	}
	if idVal := strings.TrimSpace(ctx.Query("id")); idVal != "" {
		authorName = idVal
	}

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetAuthorBooks(reqCtx, serverURL, basePath, authorName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate author books OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetSeriesCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetSeriesCatalog(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate series OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetSeriesBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	seriesName := unescapeParam(ctx.Params("name"))
	if queryVal := strings.TrimSpace(ctx.Query("name")); queryVal != "" {
		seriesName = queryVal
	}
	if idVal := strings.TrimSpace(ctx.Query("id")); idVal != "" {
		seriesName = idVal
	}

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetSeriesBooks(reqCtx, serverURL, basePath, seriesName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate series books OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetTagsCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetTagsCatalog(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate tags OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetTagBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tagName := unescapeParam(ctx.Params("name"))
	if queryVal := strings.TrimSpace(ctx.Query("name")); queryVal != "" {
		tagName = queryVal
	}
	if idVal := strings.TrimSpace(ctx.Query("id")); idVal != "" {
		tagName = idVal
	}

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetTagBooks(reqCtx, serverURL, basePath, tagName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate tag books OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetBookCover(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := ctx.Params("id")
	filePath, err := c.opdsService.GetBookCoverPath(reqCtx, bookID, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	ctx.Set("Cache-Control", "public, max-age=86400")
	ctx.Set("X-Content-Type-Options", "nosniff")
	return ctx.SendFile(filePath)
}

func (c *OPDSController) DownloadBook(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bookID := ctx.Params("id")
	fileID := ctx.Query("file_id")
	filePath, downloadName, err := c.opdsService.GetBookFileForDownload(reqCtx, bookID, fileID, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.Download(filePath, downloadName)
}

// OPDS 2.0 Handlers

func (c *OPDSController) GetOPDS2Catalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2Catalog(reqCtx, serverURL, basePath, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 catalog")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2AllBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2AllBooks(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 books catalog")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2RecentBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2RecentBooks(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 recent additions")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2HotBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2HotBooks(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 hot books")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2RandomBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2RandomBooks(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 random books")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2AuthorsCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2AuthorsCatalog(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 authors catalog")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2AuthorBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authorName := unescapeParam(ctx.Params("name"))
	if queryVal := strings.TrimSpace(ctx.Query("name")); queryVal != "" {
		authorName = queryVal
	}
	if idVal := strings.TrimSpace(ctx.Query("id")); idVal != "" {
		authorName = idVal
	}

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2AuthorBooks(reqCtx, serverURL, basePath, authorName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 author books")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2SeriesCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2SeriesCatalog(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 series catalog")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2SeriesBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	seriesName := unescapeParam(ctx.Params("name"))
	if queryVal := strings.TrimSpace(ctx.Query("name")); queryVal != "" {
		seriesName = queryVal
	}
	if idVal := strings.TrimSpace(ctx.Query("id")); idVal != "" {
		seriesName = idVal
	}

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2SeriesBooks(reqCtx, serverURL, basePath, seriesName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 series books")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2TagsCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2TagsCatalog(reqCtx, serverURL, basePath, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 tags catalog")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2TagBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tagName := unescapeParam(ctx.Params("name"))
	if queryVal := strings.TrimSpace(ctx.Query("name")); queryVal != "" {
		tagName = queryVal
	}
	if idVal := strings.TrimSpace(ctx.Query("id")); idVal != "" {
		tagName = idVal
	}

	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2TagBooks(reqCtx, serverURL, basePath, tagName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 tag books")
	}

	return sendOPDS2JSON(ctx, feed)
}

func (c *OPDSController) GetOPDS2Search(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := ctx.Query("q", "")
	serverURL := getBaseURL(ctx, c.settingsService)
	basePath := getOPDSBasePath(ctx)
	feed, err := c.opdsService.GetOPDS2Search(reqCtx, serverURL, basePath, query, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to execute OPDS 2.0 search")
	}

	return sendOPDS2JSON(ctx, feed)
}

func GetBaseURL(ctx fiber.Ctx, settings services.SettingsService) string {
	if settings != nil {
		if configured := strings.TrimRight(strings.TrimSpace(settings.ServerURL()), "/"); configured != "" {
			return configured
		}
	}
	scheme := "http"
	if ctx.Protocol() == "https" || ctx.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := strings.TrimSpace(ctx.Host())
	host = strings.ReplaceAll(host, "\r", "")
	host = strings.ReplaceAll(host, "\n", "")
	if host != "" {
		return fmt.Sprintf("%s://%s", scheme, host)
	}
	return "http://localhost:3434"
}

func getBaseURL(ctx fiber.Ctx, settings services.SettingsService) string {
	return GetBaseURL(ctx, settings)
}

func sendXML(ctx fiber.Ctx, data any) error {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal OPDS XML")
	}

	ctx.Set(fiber.HeaderContentType, "application/atom+xml; charset=utf-8")
	return ctx.Send(append([]byte(xml.Header), xmlBytes...))
}

func sendOPDS2JSON(ctx fiber.Ctx, data any) error {
	body, err := jsonx.Marshal(data)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal OPDS 2.0 JSON")
	}
	ctx.Set(fiber.HeaderContentType, "application/opds+json; charset=utf-8")
	return ctx.Send(body)
}
