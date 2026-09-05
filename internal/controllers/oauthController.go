package controllers

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
)

type OAuthController struct {
	auth     services.AuthService
	settings services.SettingsService
}

func NewOAuthController(auth services.AuthService, settings services.SettingsService) *OAuthController {
	return &OAuthController{auth: auth, settings: settings}
}

func (h *OAuthController) OAuth2Login(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	provider := strings.ToLower(c.Params("provider"))
	redirect := c.Query("redirect")

	authURL, stateUUID, err := h.auth.BuildOAuthURL(ctx, provider, redirect)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	cookieSettings := getAuthCookieSettings(c)
	stateCookie := &fiber.Cookie{
		Name:     "oauth_state",
		Value:    stateUUID,
		Expires:  time.Now().Add(15 * time.Minute),
		MaxAge:   int((15 * time.Minute).Seconds()),
		HTTPOnly: true,
		Secure:   cookieSettings.secure,
		SameSite: cookieSettings.sameSite,
		Path:     "/",
	}
	c.Cookie(stateCookie)

	return c.Redirect().To(authURL)
}

func (h *OAuthController) OAuth2Callback(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	provider := strings.ToLower(c.Params("provider"))
	code := c.Query("code")
	stateParam := c.Query("state")
	cookieState := c.Cookies("oauth_state")

	clearAuthCookie(c, "oauth_state")

	res, err := h.auth.HandleOAuthCallback(ctx, provider, code, stateParam, cookieState)
	if err != nil {
		return c.Redirect().To("/login?error=" + url.QueryEscape(err.Error()))
	}

	setAuthCookie(c, "access_token", res.AccessToken, constants.AccessTokenDuration)
	setAuthCookie(c, "refresh_token", res.RefreshToken, constants.RefreshTokenDuration)
	setCSRFCookie(c, constants.RefreshTokenDuration)

	return c.Redirect().To(res.RedirectURL)
}
