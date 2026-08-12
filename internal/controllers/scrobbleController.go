package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

type ScrobbleController struct {
	service services.ScrobbleService
}

func NewScrobbleController(service services.ScrobbleService) *ScrobbleController {
	return &ScrobbleController{service: service}
}

func hardcoverRedirectURI(ctx fiber.Ctx) string {
	scheme := "http"
	if proto := ctx.Get(fiber.HeaderXForwardedProto); proto != "" {
		scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
	} else if ctx.Protocol() == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/api/v1/scrobble/hardcover/callback", scheme, ctx.Hostname())
}

func (c *ScrobbleController) ConnectHardcover(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	redirectURI := hardcoverRedirectURI(ctx)
	authorizeURL, err := c.service.GetHardcoverAuthorizeURL(reqCtx, userID, redirectURI)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   map[string]any{"authorize_url": authorizeURL},
	})
}

func (c *ScrobbleController) HardcoverCallback(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	code := ctx.Query("code")
	state := ctx.Query("state")
	if err := c.service.HandleHardcoverCallback(reqCtx, code, state); err != nil {
		return ctx.Redirect().To("/?hardcover=error")
	}
	return ctx.Redirect().To("/?hardcover=connected")
}

func (c *ScrobbleController) SyncHardcoverProgress(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.SyncHardcoverDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	if err := c.service.SyncHardcoverProgress(reqCtx, userID, dto.BookID, dto.Progress, getOptionalClaims(ctx)); err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{Status: true})
}