package routes

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/dtos/request"
	"novelhub/internal/middlewares"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
)

// The Mihon Komga extension appends /api/v1 itself and only attaches Basic credentials after a
// 401 (its OkHttp authenticator is challenge-driven), so returning 403 here breaks auth entirely.
func KomgaRoutes(app fiber.Router, komgaController *controllers.KomgaController, authService services.AuthService, settingsService services.SettingsService) {
	auth := func(c fiber.Ctx) error {
		email, password, ok := komgaCredentials(c)
		if !ok {
			c.Set("WWW-Authenticate", `Basic realm="NovelHub Komga"`)
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		claims, err := authService.ValidateCredentials(ctx, &request.SignInDto{Email: email, Password: password})
		if err != nil {
			c.Set("WWW-Authenticate", `Basic realm="NovelHub Komga"`)
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials"))
		}
		c.Locals("user_claims", claims)
		c.Locals("uid", claims.UId)
		return c.Next()
	}

	group := app.Group("/komga", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS), auth)

	v1 := group.Group("/api/v1")
	v1.Get("/libraries", komgaController.ListLibraries)
	v1.Get("/series", komgaController.ListSeries)
	v1.Get("/series/:seriesId", komgaController.GetSeries)
	v1.Get("/series/:seriesId/books", komgaController.ListSeriesBooks)
	v1.Get("/series/:seriesId/thumbnail", komgaController.GetSeriesThumbnail)
	v1.Get("/books/:bookId", komgaController.GetBook)
	v1.Get("/books/:bookId/pages", komgaController.ListBookPages)
	v1.Get("/books/:bookId/pages/:pageNumber", komgaController.GetBookPage)
	v1.Get("/books/:bookId/thumbnail", komgaController.GetBookThumbnail)

	// Progress sync is called by Mihon's built-in tracker, a different client from the extension.
	v2 := group.Group("/api/v2")
	v2.Get("/series/:seriesId/read-progress/tachiyomi", komgaController.GetSeriesProgress)
	v2.Put("/series/:seriesId/read-progress/tachiyomi", komgaController.UpdateSeriesProgress)
}

// X-API-Key carries "<email>:<password>": there is no separate key store.
func komgaCredentials(c fiber.Ctx) (string, string, bool) {
	if apiKey := strings.TrimSpace(c.Get("X-API-Key")); apiKey != "" {
		if email, password, found := strings.Cut(apiKey, ":"); found {
			return email, password, true
		}
		return "", "", false
	}

	authHeader := c.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", "", false
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		return "", "", false
	}
	email, password, found := strings.Cut(string(payload), ":")
	if !found {
		return "", "", false
	}
	return email, password, true
}
