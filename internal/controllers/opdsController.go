package controllers

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/config"

	"github.com/gofiber/fiber/v3"
)

type OPDSController struct {
	opdsService services.OPDSService
}

func NewOPDSController(opdsService services.OPDSService) *OPDSController {
	return &OPDSController{
		opdsService: opdsService,
	}
}

func (c *OPDSController) GetRootCatalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx)
	feed, err := c.opdsService.GetRootCatalog(reqCtx, serverURL, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetRecentBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx)
	limit := int64(50)

	feed, err := c.opdsService.GetRecentBooks(reqCtx, serverURL, limit, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func (c *OPDSController) GetOPDS2Catalog(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := getBaseURL(ctx)
	feed, err := c.opdsService.GetOPDS2Catalog(reqCtx, serverURL, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS 2.0 catalog")
	}

	ctx.Set(fiber.HeaderContentType, "application/opds+json; charset=utf-8")
	return ctx.JSON(feed)
}

func getBaseURL(ctx fiber.Ctx) string {
	if serverURL := config.GetConfigWithDefault("SERVER_URL", ""); serverURL != "" {
		return strings.TrimSuffix(serverURL, "/")
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
