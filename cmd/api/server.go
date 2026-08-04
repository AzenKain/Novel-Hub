package main

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	middleware "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	static "github.com/gofiber/fiber/v3/middleware/static"
	"github.com/rs/zerolog/log"

	"novelhub/internal/controllers"
	"novelhub/internal/dtos/response"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/routes"
	"novelhub/internal/services"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/archivebook"
	"novelhub/pkg/bookparser/audiobook"
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
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/localfs"
	"novelhub/pkg/systemgate"
	"novelhub/pkg/worker"
)

//go:embed dist/* dist/assets/*
var embeddedDist embed.FS

type FiberServer struct {
	App       *fiber.App
	JobQueue  *worker.Queue
	Scheduler interface{ Stop() }
	Restart   chan struct{}
}

// parseTrustProxy reads TRUST_PROXY: "false"/empty disables it, "true" trusts
// loopback/private/link-local proxies, anything else is a comma-separated
// allowlist of proxy IPs or CIDRs. See docs/configuration.md.
//
// Stays an env var rather than an admin setting because fiber freezes it at
// New(), before the database opens, and c.Scheme() reads it to decide whether
// the auth cookie gets Secure — you would have to sign in to fix the setting
// that breaks signing in.
func parseTrustProxy(raw string) (bool, fiber.TrustProxyConfig) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "false") {
		return false, fiber.TrustProxyConfig{}
	}
	if strings.EqualFold(raw, "true") {
		return true, fiber.TrustProxyConfig{Loopback: true, Private: true, LinkLocal: true}
	}
	config := fiber.TrustProxyConfig{}
	for _, proxy := range strings.Split(raw, ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			config.Proxies = append(config.Proxies, proxy)
		}
	}
	// A value of only separators would otherwise enable proxy trust with an empty
	// allowlist, which trusts nothing but still reads as "on".
	if len(config.Proxies) == 0 {
		return false, fiber.TrustProxyConfig{}
	}
	return true, config
}

func NewHTTPServer() *FiberServer {
	trustProxy, trustProxyConfig := parseTrustProxy(config.GetConfigWithDefault("TRUST_PROXY", "false"))

	app := fiber.New(fiber.Config{
		ServerHeader:       "novelhub-api",
		AppName:            "NovelHub API",
		BodyLimit:          constants.HardMaxUploadChunkBytes + constants.MultipartBodyOverhead,
		Concurrency:        config.GetIntConfigWithDefault("FIBER_CONCURRENCY", 0),
		ReadBufferSize:     config.GetIntConfigWithDefault("FIBER_READ_BUFFER_SIZE", 0),
		WriteBufferSize:    config.GetIntConfigWithDefault("FIBER_WRITE_BUFFER_SIZE", 0),
		ReduceMemoryUsage:  true,
		JSONEncoder:        jsonx.Marshal,
		JSONDecoder:        jsonx.Unmarshal,
		TrustProxy:         trustProxy,
		TrustProxyConfig:   trustProxyConfig,
		ProxyHeader:        fiber.HeaderXForwardedFor,
		EnableIPValidation: trustProxy,
	})

	app.Use(recover.New())

	if !config.GetBoolConfigWithDefault("DISABLE_REQUEST_LOG", true) {
		app.Use(middleware.New(middleware.Config{Logger: &log.Logger}))
	}

	app.Use(helmet.New())

	return &FiberServer{App: app, Restart: make(chan struct{}, 1)}
}

func (s *FiberServer) SetupServer(db *sql.DB, ramCache cache.Cache) {
	maintenanceGate := &systemgate.Gate{}
	s.App.Use(func(c fiber.Ctx) error {
		if !maintenanceGate.Enabled() || c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead || c.Method() == fiber.MethodOptions {
			return c.Next()
		}
		return c.Status(fiber.StatusServiceUnavailable).JSON(response.CommonResponse{Status: false, Message: "restore pending; restart NovelHub to continue"})
	})
	// The production build is embedded here and calls "/api/v1" relatively, so the
	// page and API are always same-origin and CORS never applies. Only the vite dev
	// server is a genuinely different origin.
	s.App.Use(cors.New(cors.Config{
		AllowOrigins: []string{
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
	txManager := database.NewTxManager(db)
	permissionCache := services.NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to load permission cache")
	}
	settingsService := services.NewSettingsService(settingsRepo, txManager, permissionCache)
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
	parserRegistry.Register(audiobook.New(), "mp3", "m4a", "m4b", "flac")

	featureRepo := repositories.NewFeatureRepository(db, ramCache)
	highlightRepo := repositories.NewHighlightRepository(db, ramCache)
	webhookRepo := repositories.NewWebhookRepository(db, ramCache)

	authService := services.NewAuthService(userRepo, roleRepo, txManager, settingsRepo, settingsService)
	userService := services.NewUserService(userRepo, roleRepo, settingsRepo, txManager)
	roleService := services.NewRoleService(roleRepo, permissionCache, txManager)
	jobWorkers := config.GetIntConfigWithDefault("JOB_WORKERS", 1)
	if jobWorkers < 1 {
		jobWorkers = 1
	}
	jobQueue := worker.NewQueue(jobWorkers)
	s.JobQueue = jobQueue
	bookService := services.NewBookService(bookRepo, featureRepo, libraryRepo, bookFileRepo, parserRegistry, txManager, settingsService, permissionCache, jobQueue)
	libraryService := services.NewLibraryService(libraryRepo, bookRepo, bookFileRepo, parserRegistry, permissionCache, jobQueue)
	featureService := services.NewFeatureService(featureRepo, bookRepo, settingsService, permissionCache, txManager)
	highlightService := services.NewHighlightService(highlightRepo, bookRepo, permissionCache)
	metadataService := services.NewMetadataService(bookRepo)
	jobService := services.NewJobService(jobRepo, jobQueue)
	jobQueue.SetLifecycle(jobService)
	if err := jobService.Recover(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to recover background jobs")
	}
	jobScheduleRepo := repositories.NewJobScheduleRepository(db, ramCache)
	jobScheduleService := services.NewJobScheduleService(jobScheduleRepo, jobService)
	s.Scheduler = jobScheduleService
	maintenanceService := services.NewMaintenanceService(bookRepo, bookFileRepo, parserRegistry, txManager)
	dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
	logService := services.NewSystemLogService(filepath.Join(dataDir, "logs"))
	backupService := services.NewBackupService(db, dataDir, config.GetBoolConfigWithDefault("RESTORE_AUTO_RESTART", false), func() {
		select {
		case s.Restart <- struct{}{}:
		default:
		}
	}, maintenanceGate)
	calibreService := services.NewCalibreSyncService(bookRepo, bookFileRepo, txManager)
	calibreController := controllers.NewCalibreController(calibreService)
	webhookService := services.NewWebhookService(webhookRepo, jobQueue)
	webhookController := controllers.NewWebhookController(webhookService)
	bookService.SetWebhookService(webhookService)
	uploadService := services.NewUploadService(libraryService, bookService, libraryRepo, permissionCache, settingsService)

	authController := controllers.NewAuthController(authService)
	userController := controllers.NewUserController(userService)
	roleController := controllers.NewRoleController(roleService)
	bookController := controllers.NewBookController(bookService, featureService, settingsService, permissionCache)
	libraryController := controllers.NewLibraryController(libraryService)
	jobController := controllers.NewJobController(jobService, jobScheduleService)
	systemOperationsController := controllers.NewSystemOperationsController(logService, backupService)
	readerController := controllers.NewReaderController(bookService, settingsService, permissionCache)
	featureController := controllers.NewFeatureController(featureService, bookService, settingsService, permissionCache)
	highlightController := controllers.NewHighlightController(highlightService)
	metadataController := controllers.NewMetadataController(metadataService)
	settingsController := controllers.NewSettingsController(settingsService)
	uploadController := controllers.NewUploadController(uploadService)

	jobQueue.RegisterHandler("extract_metadata", func(ctx context.Context, jobID string, payload string) error {
		err := bookService.ExtractMetadata(ctx, payload)
		if err == nil {
			if enqueueErr := jobQueue.Enqueue(ctx, worker.Job{
				ID:      uuid.Must(uuid.NewV7()).String(),
				Type:    "index_book",
				Payload: payload,
			}); enqueueErr != nil {
				return enqueueErr
			}
			if book, getErr := bookService.GetBook(ctx, payload); getErr == nil && book != nil {
				webhookService.DispatchEvent(ctx, "book.created", services.BuildBookWebhookPayload(book))
			}
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
	jobQueue.RegisterHandler("clean_empty_book_dirs", func(ctx context.Context, jobID string, payload string) error {
		return maintenanceService.CleanEmptyBookDirs(ctx)
	})
	jobQueue.RegisterHandler("clean_orphan_uploads", func(ctx context.Context, jobID string, payload string) error {
		return maintenanceService.CleanOrphanUploads(ctx)
	})
	jobQueue.RegisterHandler("prune_finished_jobs", func(ctx context.Context, jobID string, payload string) error {
		return jobService.PruneFinishedJobs(ctx)
	})
	jobQueue.RegisterHandler("database_health_check", func(ctx context.Context, jobID string, payload string) error {
		return maintenanceService.CheckDatabaseHealth(ctx)
	})
	jobQueue.RegisterHandler("database_backup", func(ctx context.Context, jobID string, payload string) error {
		_, err := backupService.Create(ctx, false)
		return err
	})
	jobQueue.RegisterHandler("database_books_backup", func(ctx context.Context, jobID string, payload string) error {
		_, err := backupService.Create(ctx, true)
		return err
	})
	jobQueue.RegisterHandler("scan_library_inbox", func(ctx context.Context, jobID string, payload string) error {
		_, err := libraryService.ScanInbox(ctx)
		return err
	})

	jobQueue.RegisterHandler("webhook.dispatch", func(ctx context.Context, jobID string, payload string) error {
		var jobData struct {
			WebhookID string `json:"webhook_id"`
			EventType string `json:"event_type"`
			Data      string `json:"data"`
		}
		if err := jsonx.UnmarshalString(payload, &jobData); err != nil {
			return err
		}
		return webhookService.ExecuteDispatch(ctx, jobData.WebhookID, jobData.EventType, []byte(jobData.Data))
	})

	jobQueue.Start()
	jobScheduleService.Start()

	s.App.Use("/public", static.New(publicDir))
	s.App.Get("/storage/books/:bookID/:filename", middlewares.OptionalJwtAccess(userRepo), func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		book, err := bookService.GetBook(ctx, c.Params("bookID"))
		claims, _ := c.Locals("user_claims").(*response.JWTClaims)
		if err != nil || book == nil || !bookService.CanReadBook(ctx, book, claims) {
			return fiber.ErrForbidden
		}
		if book.CoverURL == nil || filepath.Base(*book.CoverURL) != c.Params("filename") {
			return fiber.ErrNotFound
		}
		ext := strings.ToLower(filepath.Ext(c.Params("filename")))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".ico", ".svg":
		default:
			return fiber.ErrForbidden
		}
		safePath, err := localfs.SafeJoin(booksDir, c.Params("bookID"), c.Params("filename"))
		if err != nil {
			return fiber.ErrForbidden
		}
		c.Set("X-Content-Type-Options", "nosniff")
		return c.SendFile(safePath)
	})

	api := s.App.Group("/api", middlewares.RequestBodyLimit(settingsService))
	v1 := api.Group("/v1")

	v1.Get("/health", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "ok"})
	})

	routes.AuthRoutes(v1, authController, userRepo, settingsService)
	routes.UserRoutes(v1, userController, userRepo, permissionCache)
	routes.RoleRoutes(v1, roleController, userRepo, permissionCache)
	routes.BookRoutes(v1, bookController, userRepo, bookRepo, permissionCache)
	routes.LibraryRoutes(v1, libraryController, userRepo, permissionCache)
	routes.JobRoutes(v1, jobController, userRepo, permissionCache)
	routes.SystemOperationsRoutes(v1, systemOperationsController, userRepo, permissionCache)
	routes.SetupReaderRoutes(v1, readerController, userRepo, bookRepo, permissionCache)
	routes.FeatureRoutes(v1, featureController, highlightController, userRepo, bookRepo, permissionCache)
	routes.RegisterMetadataRoutes(v1, metadataController, userRepo)
	routes.SettingsRoutes(v1, settingsController, userRepo, permissionCache)
	routes.WebhookRoutes(v1, webhookController, userRepo, permissionCache)
	routes.SetupUploadRoutes(v1, uploadController, userRepo)
	v1.Post("/calibre/import", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermCalibreSync), calibreController.ImportCalibre)

	opdsService := services.NewOPDSService(bookService, permissionCache)
	opdsController := controllers.NewOPDSController(opdsService)
	routes.OPDSRoutes(api, opdsController, authService, settingsService, userRepo)

	koboRepo := repositories.NewKoboRepository(db, ramCache)
	koboService := services.NewKoboService(bookRepo, bookFileRepo, koboRepo, bookService, featureService, permissionCache)
	koboAuthService := services.NewKoboAuthService(koboRepo)
	koboController := controllers.NewKoboController(koboService, koboAuthService)
	routes.KoboRoutes(s.App, koboController, koboRepo, userRepo, permissionCache, settingsService)
	routes.KoboSetupRoutes(v1, koboController, userRepo, permissionCache)

	syncService := services.NewSyncService(featureService, bookService, permissionCache)
	syncController := controllers.NewSyncController(syncService)
	routes.SyncRoutes(api, syncController, userRepo, permissionCache)

	trackerRepo := repositories.NewTrackerRepository(db, ramCache)
	trackerService := services.NewTrackerService(trackerRepo)
	trackerController := controllers.NewTrackerController(trackerService, featureService)
	routes.TrackerRoutes(v1, trackerController, userRepo, permissionCache, settingsService)

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
