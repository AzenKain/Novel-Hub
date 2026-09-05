package controllers

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"

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
	feed, err := c.opdsService.GetRootCatalog(reqCtx, serverURL, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetOpenSearchDescription(ctx fiber.Ctx) error {
	serverURL := getBaseURL(ctx, c.settingsService)
	desc := c.opdsService.GetOpenSearchDescription(serverURL)
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
	feed, err := c.opdsService.SearchBooksOPDS(reqCtx, serverURL, query, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to execute OPDS search")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetRecentBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)

	feed, err := c.opdsService.GetRecentBooks(reqCtx, serverURL, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetAuthorsCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	feed, err := c.opdsService.GetAuthorsCatalog(reqCtx, serverURL, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate authors OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetAuthorBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authorName := ctx.Params("name")
	serverURL := getBaseURL(ctx, c.settingsService)
	feed, err := c.opdsService.GetAuthorBooks(reqCtx, serverURL, authorName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate author books OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetSeriesCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	feed, err := c.opdsService.GetSeriesCatalog(reqCtx, serverURL, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate series OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetSeriesBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	seriesName := ctx.Params("name")
	serverURL := getBaseURL(ctx, c.settingsService)
	feed, err := c.opdsService.GetSeriesBooks(reqCtx, serverURL, seriesName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate series books OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetTagsCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	feed, err := c.opdsService.GetTagsCatalog(reqCtx, serverURL, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate tags OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetTagBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tagName := ctx.Params("name")
	serverURL := getBaseURL(ctx, c.settingsService)
	feed, err := c.opdsService.GetTagBooks(reqCtx, serverURL, tagName, opdsPageQuery(ctx), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate tag books OPDS catalog")
	}
	return sendXML(ctx, feed)
}

func (c *OPDSController) GetOPDS2Catalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx, c.settingsService)
	feed, err := c.opdsService.GetOPDS2Catalog(reqCtx, serverURL, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 catalog")
	}

	ctx.Set(fiber.HeaderContentType, "application/opds+json; charset=utf-8")
	return ctx.JSON(feed)
}

func getBaseURL(ctx fiber.Ctx, settings services.SettingsService) string {
	if settings != nil {
		if configured := settings.ServerURL(); configured != "" {
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
	return fmt.Sprintf("%s://%s", scheme, host)
}

func sendXML(ctx fiber.Ctx, data any) error {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal OPDS XML")
	}

	ctx.Set(fiber.HeaderContentType, "application/atom+xml; charset=utf-8")
	return ctx.Send(append([]byte(xml.Header), xmlBytes...))
}
