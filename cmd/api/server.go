package main

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	middleware "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	static "github.com/gofiber/fiber/v3/middleware/static"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"novelhub/internal/controllers"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/internal/routes"
	"novelhub/internal/services"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/archivebook"
	"novelhub/pkg/bookparser/comic"
	docparser "novelhub/pkg/bookparser/doc"
	"novelhub/pkg/bookparser/docx"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/bookparser/fb2"
	"novelhub/pkg/bookparser/htmlfile"
	"novelhub/pkg/bookparser/mobi"
	"novelhub/pkg/bookparser/odt"
	"novelhub/pkg/bookparser/pdf"
	"novelhub/pkg/bookparser/plain"
	"novelhub/pkg/bookparser/rtf"
	"novelhub/pkg/cache"
	"novelhub/pkg/config"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/worker"
)

//go:embed dist/* dist/assets/*
var embeddedDist embed.FS

type FiberServer struct {
	App      *fiber.App
	JobQueue *worker.Queue
}

func NewHTTPServer() *FiberServer {
	app := fiber.New(fiber.Config{
		ServerHeader:      "novelhub-api",
		AppName:           "NovelHub API",
		BodyLimit:         config.GetIntConfigWithDefault("FIBER_BODY_LIMIT", 1024*1024*1024),
		Concurrency:       config.GetIntConfigWithDefault("FIBER_CONCURRENCY", 0),
		ReadBufferSize:    config.GetIntConfigWithDefault("FIBER_READ_BUFFER_SIZE", 0),
		WriteBufferSize:   config.GetIntConfigWithDefault("FIBER_WRITE_BUFFER_SIZE", 0),
		ReduceMemoryUsage: true,
		JSONEncoder:       jsonx.Marshal,
		JSONDecoder:       jsonx.Unmarshal,
	})

	if !config.GetBoolConfigWithDefault("DISABLE_REQUEST_LOG", false) {
		logger := zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
		app.Use(middleware.New(middleware.Config{Logger: &logger}))
	}

	app.Use(helmet.New())

	return &FiberServer{App: app}
}

func (s *FiberServer) SetupServer(db *sql.DB, ramCache cache.Cache) {
	frontendURL := config.GetConfigWithDefault("FRONTEND_URL", "http://localhost:5173")
	s.App.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			frontendURL,
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "Origin", "X-Requested-With"},
		AllowCredentials: true,
	}))

	if !config.GetBoolConfigWithDefault("DISABLE_RESPONSE_COMPRESSION", false) {
		s.App.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	}

	booksDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "books")
	os.MkdirAll(booksDir, 0755)

	publicDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "public")
	os.MkdirAll(publicDir, 0755)

	bookFileRepo, err := repositories.NewBookFileRepository(booksDir)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize book file repository")
	}

	userRepo := repositories.NewUserRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	libraryRepo := repositories.NewLibraryRepository(db, ramCache)
	jobRepo := repositories.NewJobRepository(db, ramCache)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	permissionCache := services.NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to load permission cache")
	}
	settingsService := services.NewSettingsService(settingsRepo)
	if err := settingsService.Reload(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to load settings cache")
	}

	parserRegistry := bookparser.NewRegistry()
	parserRegistry.Register(epub.NewParser(), "epub", "kepub.epub")
	parserRegistry.Register(plain.NewParser(), "txt", "md", "markdown")
	parserRegistry.Register(htmlfile.NewParser(), "html", "htm")
	parserRegistry.Register(docx.NewParser(), "docx")
	parserRegistry.Register(docparser.NewParser(), "doc")
	parserRegistry.Register(odt.NewParser(), "odt")
	parserRegistry.Register(rtf.NewParser(), "rtf")
	parserRegistry.Register(fb2.NewParser(), "fb2")
	parserRegistry.Register(pdf.NewParser(), "pdf")
	parserRegistry.Register(mobi.NewParser(), "mobi", "azw", "azw3", "amz")
	parserRegistry.Register(archivebook.NewParser("zip"), "zip")
	parserRegistry.Register(archivebook.NewParser("fbz"), "fbz")
	parserRegistry.Register(comic.NewParser("cbz"), "cbz")
	parserRegistry.Register(comic.NewParser("cbr"), "cbr")
	parserRegistry.Register(comic.NewParser("cbt"), "cbt")
	parserRegistry.Register(comic.NewParser("cb7"), "cb7")

	featureRepo := repositories.NewFeatureRepository(db, ramCache)
	highlightRepo := repositories.NewHighlightRepository(db, ramCache)
	txManager := database.NewTxManager(db)

	authService := services.NewAuthService(userRepo, roleRepo, txManager, settingsRepo, settingsService)
	userService := services.NewUserService(userRepo, roleRepo, txManager)
	roleService := services.NewRoleService(roleRepo, permissionCache, txManager)
	bookService := services.NewBookService(bookRepo, bookFileRepo, parserRegistry, txManager, settingsService, permissionCache)
	jobWorkers := config.GetIntConfigWithDefault("JOB_WORKERS", 1)
	if jobWorkers < 1 {
		jobWorkers = 1
	}
	jobQueue := worker.NewQueue(jobWorkers)
	s.JobQueue = jobQueue
	libraryService := services.NewLibraryService(libraryRepo, bookRepo, bookFileRepo, jobQueue)
	featureService := services.NewFeatureService(featureRepo, bookRepo, settingsService, permissionCache, txManager)
	highlightService := services.NewHighlightService(highlightRepo)
	metadataService := services.NewMetadataService(bookRepo)
	jobService := services.NewJobService(jobRepo)
	maintenanceService := services.NewMaintenanceService(bookRepo, bookFileRepo, parserRegistry, txManager)

	authController := controllers.NewAuthController(authService)
	userController := controllers.NewUserController(userService)
	roleController := controllers.NewRoleController(roleService)
	bookController := controllers.NewBookController(bookService, featureService, settingsService, permissionCache)
	libraryController := controllers.NewLibraryController(libraryService)
	jobController := controllers.NewJobController(jobService)
	readerController := controllers.NewReaderController(bookService, settingsService, permissionCache)
	featureController := controllers.NewFeatureController(featureService, bookService, settingsService, permissionCache)
	highlightController := controllers.NewHighlightController(highlightService)
	metadataController := controllers.NewMetadataController(metadataService)
	settingsController := controllers.NewSettingsController(settingsService)

	jobQueue.RegisterHandler("extract_metadata", func(ctx context.Context, jobID string, payload string) error {
		err := bookService.ExtractMetadata(ctx, payload)
		if err == nil {
			jobQueue.Enqueue(worker.Job{
				ID:      uuid.Must(uuid.NewV7()).String(),
				Type:    "index_book",
				Payload: payload,
			})
		}
		return err
	})

	jobQueue.RegisterHandler("hash_file", func(ctx context.Context, jobID string, payload string) error {
		return maintenanceService.HashFile(ctx, payload)
	})

	jobQueue.RegisterHandler("index_book", func(ctx context.Context, jobID string, payload string) error {
		return maintenanceService.IndexBook(ctx, payload)
	})

	jobQueue.RegisterHandler("maintenance", func(ctx context.Context, jobID string, payload string) error {
		return maintenanceService.RunMaintenance(ctx)
	})

	jobQueue.Start()

	s.App.Use("/storage/books", static.New(booksDir))
	s.App.Use("/public", static.New(publicDir))
	s.App.Get("/books/:bookID/:filename", func(c fiber.Ctx) error {
		return c.Redirect().Status(fiber.StatusMovedPermanently).To(
			"/storage/books/" + c.Params("bookID") + "/" + c.Params("filename"),
		)
	})

	api := s.App.Group("/api")
	v1 := api.Group("/v1")

	v1.Get("/health", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "ok"})
	})

	routes.AuthRoutes(v1, authController, userRepo)
	routes.UserRoutes(v1, userController, userRepo, permissionCache)
	routes.RoleRoutes(v1, roleController, userRepo, permissionCache)
	routes.BookRoutes(v1, bookController, userRepo, bookRepo, permissionCache)
	routes.LibraryRoutes(v1, libraryController, userRepo, permissionCache)
	routes.JobRoutes(v1, jobController, userRepo, permissionCache)
	routes.SetupReaderRoutes(v1, readerController, userRepo, bookRepo, permissionCache)
	routes.FeatureRoutes(v1, featureController, highlightController, userRepo, bookRepo, permissionCache)
	routes.RegisterMetadataRoutes(v1, metadataController, userRepo)
	routes.SettingsRoutes(v1, settingsController, userRepo, permissionCache)

	serveEmbeddedFrontend(s.App)
	routes.NotFoundRoute(s.App)
}

func serveEmbeddedFrontend(app *fiber.App) {
	dist, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return
	}

	serveIndex := func(c fiber.Ctx) error {
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			return c.Next()
		}
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.Status(fiber.StatusOK).Send(index)
	}

	app.Get("/", serveIndex)
	app.Get("/index.html", serveIndex)

	app.Use(static.New("", static.Config{
		FS:     dist,
		MaxAge: 31536000, // 1 year for assets
		NotFoundHandler: func(c fiber.Ctx) error {
			if c.Method() != fiber.MethodGet {
				return c.Next()
			}
			return serveIndex(c)
		},
	}))
}
