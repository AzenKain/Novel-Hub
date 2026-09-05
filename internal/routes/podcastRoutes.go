package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func PodcastRoutes(app fiber.Router, podcastController *controllers.PodcastController, userRepo repositories.UserRepository, podcastRepo repositories.PodcastRepository, permissionCache services.PermissionCache) {
	group := app.Group("/podcasts", middlewares.JwtAccess(userRepo))

	group.Get("/", middlewares.RequirePermission(permissionCache, constants.PermPodcastManage), podcastController.ListPodcasts)
	group.Post("/", middlewares.RequirePermission(permissionCache, constants.PermPodcastManage, middlewares.LibraryIDBody()), podcastController.Subscribe)
	group.Put("/:id", middlewares.RequirePermission(permissionCache, constants.PermPodcastManage, middlewares.PodcastLibraryAttr(podcastRepo, "id")), podcastController.UpdatePodcast)
	group.Delete("/:id", middlewares.RequirePermission(permissionCache, constants.PermPodcastManage, middlewares.PodcastLibraryAttr(podcastRepo, "id")), podcastController.DeletePodcast)
	group.Post("/:id/refresh", middlewares.RequirePermission(permissionCache, constants.PermPodcastManage, middlewares.PodcastLibraryAttr(podcastRepo, "id")), podcastController.RefreshPodcast)
	group.Get("/:id/episodes", middlewares.RequirePermission(permissionCache, constants.PermPodcastManage, middlewares.PodcastLibraryAttr(podcastRepo, "id")), podcastController.ListEpisodes)
	group.Post("/:id/episodes/:episode_id/download", middlewares.RequirePermission(permissionCache, constants.PermPodcastManage, middlewares.PodcastLibraryAttr(podcastRepo, "id")), podcastController.DownloadEpisode)
}
