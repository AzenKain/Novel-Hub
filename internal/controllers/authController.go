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
	"novelhub/pkg/constants"
	"novelhub/pkg/crypto"
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
}

// Derived from the request, not configured: c.Scheme() is "https" on TLS or when
// a trusted proxy says so (TRUST_PROXY), which is exactly when Secure is needed.
// Domain is never set, so the cookie stays scoped to the host that served it.
//
// SameSite=Lax is the only CSRF defence this app has — there is no token or
// origin check anywhere — so it must not be loosened to None.
func getAuthCookieSettings(c fiber.Ctx) authCookieSettings {
	return authCookieSettings{
		secure:   c.Scheme() == "https",
		sameSite: "Lax",
	}
}

func setAuthCookie(c fiber.Ctx, name string, value string, duration time.Duration) {
	settings := getAuthCookieSettings(c)
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
	c.Cookie(cookie)
}

func setCSRFCookie(c fiber.Ctx, duration time.Duration) {
	token, err := crypto.GenerateRandomHex(16)
	if err != nil {
		return
	}
	settings := getAuthCookieSettings(c)
	cookie := &fiber.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Expires:  time.Now().Add(duration),
		MaxAge:   int(duration.Seconds()),
		HTTPOnly: false,
		Secure:   settings.secure,
		SameSite: settings.sameSite,
		Path:     "/",
	}
	c.Cookie(cookie)
}


func clearAuthCookie(c fiber.Ctx, name string) {
	settings := getAuthCookieSettings(c)
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

func (h *AuthController) RequestOTP(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	dto := &request.RequestOTPDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.RequestOTP(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *AuthController) VerifyOTP(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.VerifyOTPDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.VerifyOTP(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *AuthController) ResetPasswordWithOTP(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dto := &request.ResetPasswordWithOTPDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	if err := h.service.ResetPasswordWithOTP(ctx, dto); err != nil {
		return apperrors.HandleError(c, err)
	}
	clearAuthCookie(c, "access_token")
	clearAuthCookie(c, "refresh_token")
	clearAuthCookie(c, "csrf_token")
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "Password updated successfully"})
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
	if res.TOTPRequired {
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
	}
	setAuthCookie(c, "access_token", res.AccessToken, constants.AccessTokenDuration)
	setAuthCookie(c, "refresh_token", res.RefreshToken, constants.RefreshTokenDuration)
	setCSRFCookie(c, constants.RefreshTokenDuration)

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
		clearAuthCookie(c, "csrf_token")
		return apperrors.HandleError(c, err)
	}
	setAuthCookie(c, "access_token", res.AccessToken, constants.AccessTokenDuration)
	setAuthCookie(c, "refresh_token", res.RefreshToken, constants.RefreshTokenDuration)
	setCSRFCookie(c, constants.RefreshTokenDuration)

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
	clearAuthCookie(c, "csrf_token")
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "Logged out successfully"})
}
