package controllers

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"

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
	feed, err := c.opdsService.GetRootCatalog(reqCtx, serverURL)
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

	feed, err := c.opdsService.GetRecentBooks(reqCtx, serverURL, limit)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to generate OPDS catalog")
	}

	return sendXML(ctx, feed)
}

func getBaseURL(ctx fiber.Ctx) string {
	scheme := "http"
	if ctx.Protocol() == "https" || ctx.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, ctx.Host())
}

func sendXML(ctx fiber.Ctx, data interface{}) error {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal OPDS XML")
	}

	ctx.Set(fiber.HeaderContentType, "application/atom+xml; charset=utf-8")
	return ctx.Send(append([]byte(xml.Header), xmlBytes...))
}
