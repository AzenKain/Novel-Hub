package controllers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"novelhub/pkg/apperrors"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/validator"
)

type AuthController struct {
	service services.AuthService
}

func NewAuthController(svc services.AuthService) *AuthController {
	return &AuthController{service: svc}
}

type authCookieSettings struct {
	secure   bool
	sameSite string
	domain   string
}

func getAuthCookieSettings() authCookieSettings {
	secure := config.GetBoolConfigWithDefault("COOKIE_SECURE", false)
	sameSite := config.GetConfigWithDefault("COOKIE_SAMESITE", "Lax")
	if sameSite == "" {
		sameSite = "Lax"
	}
	return authCookieSettings{
		secure:   secure,
		sameSite: sameSite,
		domain:   config.GetConfigWithDefault("COOKIE_DOMAIN", ""),
	}
}

func setAuthCookie(c fiber.Ctx, name string, value string, duration time.Duration) {
	settings := getAuthCookieSettings()
	cookie := &fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(duration),
		MaxAge:   int(duration.Seconds()),
		HTTPOnly: true,
		Secure:   settings.secure,
		SameSite: settings.sameSite,
		Path:     "/",
	}
	if settings.domain != "" {
		cookie.Domain = settings.domain
	}
	c.Cookie(cookie)
}

func clearAuthCookie(c fiber.Ctx, name string) {
	settings := getAuthCookieSettings()
	cookie := &fiber.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   settings.secure,
		SameSite: settings.sameSite,
		Path:     "/",
	}
	if settings.domain != "" {
		cookie.Domain = settings.domain
	}
	c.Cookie(cookie)
}

func (h *AuthController) Register(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.RegisterDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.Register(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *AuthController) SubmitSetup(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dto := &request.SetupDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.SubmitSetup(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *AuthController) Signin(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SignInDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.Signin(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	setAuthCookie(c, "access_token", res.AccessToken, constants.AccessTokenDuration)
	setAuthCookie(c, "refresh_token", res.RefreshToken, constants.RefreshTokenDuration)

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *AuthController) getRefreshToken(c fiber.Ctx) string {
	if cookie := c.Cookies("refresh_token"); cookie != "" {
		return cookie
	}
	auth := c.Get("Authorization")
	if auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func (h *AuthController) RefreshToken(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token := h.getRefreshToken(c)
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Missing refresh token"})
	}

	uid, ok := c.Locals("uid").(string)
	if !ok || uid == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.RefreshToken(ctx, uid, token)
	if err != nil {
		clearAuthCookie(c, "access_token")
		clearAuthCookie(c, "refresh_token")
		return apperrors.HandleError(c, err)
	}
	setAuthCookie(c, "access_token", res.AccessToken, constants.AccessTokenDuration)
	setAuthCookie(c, "refresh_token", res.RefreshToken, constants.RefreshTokenDuration)

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *AuthController) Logout(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uid, ok := c.Locals("uid").(string)
	if !ok || uid == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	err := h.service.Logout(ctx, uid)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	clearAuthCookie(c, "access_token")
	clearAuthCookie(c, "refresh_token")
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "Logged out successfully"})
}
