package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/webdav"
)

type WebDAVController struct {
	service services.WebDAVService
}

func NewWebDAVController(service services.WebDAVService) *WebDAVController {
	return &WebDAVController{service: service}
}

// HandleOptions responds to WebDAV OPTIONS discovery requests.
func (h *WebDAVController) HandleOptions(c fiber.Ctx) error {
	c.Set("DAV", "1, 2")
	c.Set("MS-Author-Via", "DAV")
	c.Set("Allow", "OPTIONS, GET, HEAD, PROPFIND")
	return c.SendStatus(fiber.StatusOK)
}

// HandlePropfind handles WebDAV PROPFIND requests to list directories and file metadata.
func (h *WebDAVController) HandlePropfind(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	claims := getOptionalClaims(c)
	depth := webdav.ParseDepth(c.Get("Depth"))
	reqPath := c.Path()

	nodes, err := h.service.ResolvePath(ctx, reqPath, claims, depth)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	xmlData, err := webdav.BuildMultiStatusXML(nodes)
	if err != nil {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to generate WebDAV XML: %v", err)))
	}

	c.Set("Content-Type", "application/xml; charset=utf-8")
	c.Status(207)
	return c.Send(xmlData)
}

// HandleGet handles downloading a book file or viewing a directory index.
func (h *WebDAVController) HandleGet(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	claims := getOptionalClaims(c)
	reqPath := c.Path()

	filePath, mimeType, downloadName, err := h.service.GetBookFile(ctx, reqPath, claims)
	if err != nil {
		nodes, resolveErr := h.service.ResolvePath(ctx, reqPath, claims, 1)
		if resolveErr == nil && len(nodes) > 0 {
			xmlData, _ := webdav.BuildMultiStatusXML(nodes)
			c.Set("Content-Type", "application/xml; charset=utf-8")
			c.Status(207)
			return c.Send(xmlData)
		}
		return apperrors.HandleError(c, err)
	}

	c.Set("Content-Type", mimeType)
	c.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, downloadName))
	return c.SendFile(filePath)
}

// HandleHead responds to HEAD requests for a WebDAV resource.
func (h *WebDAVController) HandleHead(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims := getOptionalClaims(c)
	reqPath := c.Path()

	filePath, mimeType, downloadName, err := h.service.GetBookFile(ctx, reqPath, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	c.Set("Content-Type", mimeType)
	c.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, downloadName))
	return c.SendFile(filePath)
}
