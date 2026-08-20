package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func CustomizationRoutes(
	r fiber.Router,
	ctrl *controllers.CustomizationController,
	userRepo repositories.UserRepository,
	permissionCache services.PermissionCache,
) {
	// Soundscapes
	r.Get("/soundscapes", middlewares.OptionalJwtAccess(userRepo), ctrl.ListSoundscapes)
	r.Get("/soundscapes/:id/stream", middlewares.OptionalJwtAccess(userRepo), ctrl.StreamSoundscape)
	r.Post("/soundscapes/upload", middlewares.JwtAccess(userRepo), middlewares.RequireAnyPermission(permissionCache, constants.PermUserSoundscapeManage, constants.PermAdminSoundscapeManage), ctrl.UploadSoundscape)
	r.Delete("/soundscapes/:id", middlewares.JwtAccess(userRepo), middlewares.RequireAnyPermission(permissionCache, constants.PermUserSoundscapeManage, constants.PermAdminSoundscapeManage), ctrl.DeleteSoundscape)

	// Custom Fonts
	r.Get("/fonts", middlewares.OptionalJwtAccess(userRepo), ctrl.ListCustomFonts)
	r.Get("/fonts/:id/file", middlewares.OptionalJwtAccess(userRepo), ctrl.ServeFontFile)
	r.Post("/fonts/upload", middlewares.JwtAccess(userRepo), middlewares.RequireAnyPermission(permissionCache, constants.PermUserFontManage, constants.PermAdminFontManage), ctrl.UploadCustomFont)
	r.Delete("/fonts/:id", middlewares.JwtAccess(userRepo), middlewares.RequireAnyPermission(permissionCache, constants.PermUserFontManage, constants.PermAdminFontManage), ctrl.DeleteCustomFont)

	// Custom Themes
	r.Get("/themes", middlewares.OptionalJwtAccess(userRepo), ctrl.ListCustomThemes)
	r.Get("/themes/:id", middlewares.OptionalJwtAccess(userRepo), ctrl.GetCustomTheme)
	r.Post("/themes", middlewares.JwtAccess(userRepo), middlewares.RequireAnyPermission(permissionCache, constants.PermUserThemeManage, constants.PermAdminThemeManage), ctrl.CreateCustomTheme)
	r.Put("/themes/:id", middlewares.JwtAccess(userRepo), middlewares.RequireAnyPermission(permissionCache, constants.PermUserThemeManage, constants.PermAdminThemeManage), ctrl.UpdateCustomTheme)
	r.Delete("/themes/:id", middlewares.JwtAccess(userRepo), middlewares.RequireAnyPermission(permissionCache, constants.PermUserThemeManage, constants.PermAdminThemeManage), ctrl.DeleteCustomTheme)
}
