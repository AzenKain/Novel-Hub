package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
	"novelhub/pkg/bookparser/csv"
	docparser "novelhub/pkg/bookparser/doc"
	"novelhub/pkg/bookparser/docx"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/bookparser/fb2"
	"novelhub/pkg/bookparser/htmlfile"
	"novelhub/pkg/bookparser/mobi"
	"novelhub/pkg/bookparser/odt"
	"novelhub/pkg/bookparser/pdf"
	"novelhub/pkg/bookparser/plain"
	"novelhub/pkg/bookparser/presentation"
	"novelhub/pkg/bookparser/rtf"
	"novelhub/pkg/bookparser/spreadsheet"
	"novelhub/pkg/bookparser/tex"
	"novelhub/pkg/cache"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/localfs"
	"novelhub/pkg/systemgate"
	"novelhub/pkg/worker"
)

//go:embed all:dist
var embeddedDist embed.FS

type FiberServer struct {
	App       *fiber.App
	JobQueue  *worker.Queue
	Scheduler interface{ Stop() }
	Restart   chan struct{}
}

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

	s.App.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "Origin", "X-Requested-With", "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	s.App.Use(middlewares.CSRFProtection())

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
	parserRegistry.Register(comic.NewParser("rar"), "rar")
	parserRegistry.Register(comic.NewParser("7z"), "7z")
	parserRegistry.Register(audiobook.New(), "mp3", "m4a", "m4b", "flac")
	parserRegistry.Register(csv.NewParser(), "csv", "tsv")
	parserRegistry.Register(tex.NewParser(), "tex", "latex", "ltx")
	parserRegistry.Register(presentation.NewParser(), "pptx", "ppt", "odp")
	parserRegistry.Register(spreadsheet.NewParser(), "xlsx", "xls", "ods")

	featureRepo := repositories.NewFeatureRepository(db, ramCache)
	readListRepo := repositories.NewReadListRepository(db, ramCache)
	highlightRepo := repositories.NewHighlightRepository(db, ramCache)
	webhookRepo := repositories.NewWebhookRepository(db, ramCache)
	auditRepo := repositories.NewAuditRepository(db, ramCache)
	totpRepo := repositories.NewTOTPRepository(db, ramCache)
	deviceRepo := repositories.NewDeviceRepository(db, ramCache)

	authService := services.NewAuthService(userRepo, roleRepo, txManager, settingsRepo, settingsService, services.NewOTPStore(ramCache))
	userService := services.NewUserService(userRepo, roleRepo, settingsRepo, txManager, permissionCache, settingsService)
	roleService := services.NewRoleService(roleRepo, permissionCache, txManager)
	auditService := services.NewAuditService(auditRepo, userRepo)
	totpService := services.NewTOTPService(totpRepo, ramCache)
	authService.SetTOTPService(totpService)
	jobWorkers := config.GetIntConfigWithDefault("JOB_WORKERS", 1)
	if jobWorkers < 1 {
		jobWorkers = 1
	}
	jobQueue := worker.NewQueue(jobWorkers)
	s.JobQueue = jobQueue
	assetCache, err := cache.NewByteCache(0, constants.NormalCacheDuration)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build asset cache")
	}
	bookService := services.NewBookService(bookRepo, featureRepo, libraryRepo, bookFileRepo, parserRegistry, txManager, settingsService, permissionCache, jobQueue, assetCache)
	libraryService := services.NewLibraryService(libraryRepo, bookRepo, bookFileRepo, parserRegistry, permissionCache, settingsService, jobQueue)
	featureService := services.NewFeatureService(featureRepo, bookRepo, settingsService, permissionCache, txManager)
	readListService := services.NewReadListService(readListRepo, bookRepo, bookService, txManager)
	highlightService := services.NewHighlightService(highlightRepo, bookRepo, permissionCache)
	metadataService := services.NewMetadataService(bookRepo, libraryService)
	jobService := services.NewJobService(jobRepo, jobQueue)
	jobQueue.SetLifecycle(jobService)
	if err := jobService.Recover(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to recover background jobs")
	}
	jobScheduleRepo := repositories.NewJobScheduleRepository(db, ramCache)
	jobScheduleService := services.NewJobScheduleService(jobScheduleRepo, jobService)
	s.Scheduler = jobScheduleService
	magicCodeRepo := repositories.NewMagicCodeRepository(db, ramCache)
	maintenanceService := services.NewMaintenanceService(bookRepo, bookFileRepo, magicCodeRepo, parserRegistry, txManager)
	dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
	logService := services.NewSystemLogService(filepath.Join(dataDir, "logs"))
	backupService := services.NewBackupService(db, dataDir, config.GetBoolConfigWithDefault("RESTORE_AUTO_RESTART", false), func() {
		select {
		case s.Restart <- struct{}{}:
		default:
		}
	}, maintenanceGate)
	calibreService := services.NewCalibreSyncService(bookRepo, bookFileRepo, txManager, settingsService)
	calibreController := controllers.NewCalibreController(calibreService)
	webhookService := services.NewWebhookService(webhookRepo, jobQueue, settingsService)
	webhookController := controllers.NewWebhookController(webhookService)
	bookService.SetWebhookService(webhookService)
	featureService.SetWebhookService(webhookService)
	jobService.SetWebhookService(webhookService)
	uploadService := services.NewUploadService(libraryService, bookService, libraryRepo, permissionCache, settingsService)
	deviceService := services.NewDeviceService(deviceRepo, bookRepo, bookService, settingsService, permissionCache, jobQueue)
	magicCodeService := services.NewMagicCodeService(magicCodeRepo, userRepo, authService)
	ageRatingRepo := repositories.NewAgeRatingRepository(db, ramCache)
	ageRatingService := services.NewAgeRatingService(ageRatingRepo)
	audiobookRepo := repositories.NewAudiobookRepository(db, ramCache)
	audiobookService := services.NewAudiobookService(audiobookRepo, bookRepo, bookFileRepo, jobQueue)
	podcastRepo := repositories.NewPodcastRepository(db, ramCache)
	podcastService := services.NewPodcastService(podcastRepo, bookRepo, bookFileRepo, libraryRepo, jobQueue)

	authController := controllers.NewAuthController(authService)
	oauthController := controllers.NewOAuthController(authService, settingsService)
	magicCodeController := controllers.NewMagicCodeController(magicCodeService, settingsService)
	ageRatingController := controllers.NewAgeRatingController(ageRatingService)
	audiobookController := controllers.NewAudiobookController(audiobookService)
	podcastController := controllers.NewPodcastController(podcastService)
	userController := controllers.NewUserController(userService, auditService)
	roleController := controllers.NewRoleController(roleService, auditService)
	bookController := controllers.NewBookController(bookService, featureService, settingsService, permissionCache, auditService)
	libraryController := controllers.NewLibraryController(libraryService)
	jobController := controllers.NewJobController(jobService, jobScheduleService)
	systemOperationsController := controllers.NewSystemOperationsController(logService, backupService, auditService, ramCache)
	readerController := controllers.NewReaderController(bookService, settingsService, permissionCache)
	featureController := controllers.NewFeatureController(featureService, bookService, settingsService, permissionCache)
	smartFilterController := controllers.NewSmartFilterController(featureService, bookService)
	readListController := controllers.NewReadListController(readListService)
	highlightController := controllers.NewHighlightController(highlightService)
	metadataController := controllers.NewMetadataController(metadataService)
	settingsController := controllers.NewSettingsController(settingsService, auditService)
	auditController := controllers.NewAuditController(auditService)
	totpController := controllers.NewTOTPController(totpService, userService, auditService)
	uploadController := controllers.NewUploadController(uploadService)
	deviceController := controllers.NewDeviceController(deviceService)

	customizationRepo := repositories.NewCustomizationRepository(db, ramCache)
	customizationService := services.NewCustomizationService(customizationRepo, permissionCache, settingsService, dataDir)
	customizationController := controllers.NewCustomizationController(customizationService, settingsService)

	userService.SetJobQueue(jobQueue)
	authService.SetJobQueue(jobQueue)

	jobQueue.RegisterHandler("push_book_to_device", func(ctx context.Context, jobID string, payload string) error {
		return deviceService.ExecutePushJob(ctx, payload)
	})

	jobQueue.RegisterHandler("send_book_email", func(ctx context.Context, jobID string, payload string) error {
		return bookService.ExecuteSendBookEmailJob(ctx, payload)
	})

	jobQueue.RegisterHandler("send_user_email", func(ctx context.Context, jobID string, payload string) error {
		return userService.ExecuteSendUserEmailJob(ctx, payload)
	})

	jobQueue.RegisterHandler("send_otp_email", func(ctx context.Context, jobID string, payload string) error {
		return authService.ExecuteSendOTPJob(ctx, payload)
	})

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

	jobQueue.RegisterHandler("merge_audio", func(ctx context.Context, jobID string, payload string) error {
		return audiobookService.ExecuteMergeAudioJob(ctx, payload)
	})

	jobQueue.RegisterHandler("convert_book", func(ctx context.Context, jobID string, payload string) error {
		return bookService.ExecuteConvertBookJob(ctx, payload)
	})

	jobQueue.RegisterHandler("podcast_refresh", func(ctx context.Context, jobID string, payload string) error {
		return podcastService.ExecutePodcastRefreshJob(ctx, payload)
	})

	jobQueue.RegisterHandler("podcast_download", func(ctx context.Context, jobID string, payload string) error {
		return podcastService.ExecutePodcastDownloadJob(ctx, payload)
	})

	jobQueue.RegisterHandler("prune_audit_logs", func(ctx context.Context, jobID string, payload string) error {
		_, err := auditService.Prune(ctx)
		return err
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

	jobQueue.RegisterHandler("delete_library", func(ctx context.Context, jobID string, payload string) error {
		return libraryService.ProcessDeleteLibrary(ctx, payload)
	})

	jobQueue.RegisterHandler("scan_metadata_enrich", func(ctx context.Context, jobID string, payload string) error {
		return bookService.BatchEnrichBooks(ctx)
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

	proxyAuthMiddleware := middlewares.ProxyAuth(settingsService, authService, userRepo, roleRepo, txManager)
	api := s.App.Group("/api", middlewares.RequestBodyLimit(settingsService), proxyAuthMiddleware)
	v1 := api.Group("/v1")

	v1.Get("/health", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(response.CommonResponse{Status: false, Message: "database unavailable"})
		}
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "ok"})
	})

	routes.AuthRoutes(v1, authController, oauthController, userRepo, settingsService)
	routes.MagicCodeRoutes(v1, magicCodeController, userRepo, settingsService)
	routes.AgeRatingRoutes(v1, ageRatingController, userRepo, bookRepo, permissionCache)
	routes.AudiobookRoutes(v1, audiobookController, userRepo, bookRepo, permissionCache)
	routes.PodcastRoutes(v1, podcastController, userRepo, podcastRepo, permissionCache)
	routes.UserRoutes(v1, userController, userRepo, permissionCache)
	routes.RoleRoutes(v1, roleController, userRepo, permissionCache)
	routes.BookRoutes(v1, bookController, userRepo, bookRepo, permissionCache)
	routes.LibraryRoutes(v1, libraryController, userRepo, permissionCache)
	routes.JobRoutes(v1, jobController, userRepo, permissionCache)
	routes.SystemOperationsRoutes(v1, systemOperationsController, userRepo, permissionCache)
	routes.SetupReaderRoutes(v1, readerController, userRepo, bookRepo, permissionCache)
	routes.FeatureRoutes(v1, featureController, highlightController, readListController, userRepo, bookRepo, permissionCache)
	routes.SmartFilterRoutes(v1, smartFilterController, userRepo, permissionCache)
	routes.RegisterMetadataRoutes(v1, metadataController, userRepo)
	routes.SettingsRoutes(v1, settingsController, userRepo, permissionCache)
	routes.AuditRoutes(v1, auditController, userRepo, permissionCache)
	routes.TOTPRoutes(v1, totpController, userRepo, settingsService)
	routes.WebhookRoutes(v1, webhookController, userRepo, permissionCache)
	routes.SetupUploadRoutes(v1, uploadController, userRepo, permissionCache)
	routes.DeviceRoutes(v1, deviceController, userRepo, bookRepo, permissionCache)
	routes.CustomizationRoutes(v1, customizationController, userRepo, permissionCache)
	v1.Post("/calibre/import", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermCalibreSync), calibreController.ImportCalibre)

	opdsService := services.NewOPDSService(bookService, permissionCache)
	opdsController := controllers.NewOPDSController(opdsService, settingsService)
	routes.OPDSRoutes(api, opdsController, authService, settingsService, userRepo)

	vbookFS, _ := fs.Sub(embeddedDist, "dist/vbook")
	vbookService := services.NewVBookService(bookRepo, bookRepo, audiobookRepo, bookService, vbookFS, ramCache)
	vbookController := controllers.NewVBookController(vbookService, settingsService)
	routes.VBookRoutes(api, vbookController, authService, settingsService, userRepo)

	koboRepo := repositories.NewKoboRepository(db, ramCache)
	koboService := services.NewKoboService(bookRepo, bookFileRepo, koboRepo, readListRepo, bookService, featureService, permissionCache, ramCache)
	koboAuthService := services.NewKoboAuthService(koboRepo)
	koboController := controllers.NewKoboController(koboService, koboAuthService, settingsService)
	routes.KoboRoutes(s.App, koboController, koboRepo, userRepo, permissionCache, settingsService)
	routes.KoboSetupRoutes(v1, koboController, userRepo, permissionCache)

	// Mounted on the app root, not on api: the Mihon Komga extension appends /api/v1 to the
	// address the user types, so the user enters http://host/komga and requests land on
	// /komga/api/v1/... See internal/routes/komgaRoutes.go.
	komgaRepo := repositories.NewKomgaRepository(db, ramCache)
	komgaService := services.NewKomgaService(komgaRepo, bookRepo, bookFileRepo, readListRepo, userRepo, bookService, libraryService, featureService, permissionCache, ramCache)
	komgaController := controllers.NewKomgaController(komgaService)
	routes.KomgaRoutes(s.App, komgaController, authService, settingsService)

	syncService := services.NewSyncService(featureService, bookService, permissionCache)
	syncController := controllers.NewSyncController(syncService)
	routes.SyncRoutes(api, syncController, userRepo, permissionCache)

	trackerRepo := repositories.NewTrackerRepository(db, ramCache)
	trackerService := services.NewTrackerService(trackerRepo)
	trackerController := controllers.NewTrackerController(trackerService, featureService)
	routes.TrackerRoutes(v1, trackerController, userRepo, permissionCache, settingsService)

	integrationsService := services.NewIntegrationsService(highlightRepo, trackerRepo, bookRepo, permissionCache)
	integrationsController := controllers.NewIntegrationsController(integrationsService)
	routes.IntegrationRoutes(v1, integrationsController, userRepo, bookRepo, permissionCache)

	scrobbleService := services.NewScrobbleService(trackerRepo, bookRepo, settingsService, ramCache, permissionCache)
	scrobbleController := controllers.NewScrobbleController(scrobbleService)
	routes.ScrobbleRoutes(v1, scrobbleController, userRepo, permissionCache)

	serveEmbeddedFrontend(s.App, bookService, settingsService)
	routes.NotFoundRoute(s.App)
}

func resolveOrigin(c fiber.Ctx, settingsServerURL string) string {
	if sURL := strings.TrimRight(strings.TrimSpace(settingsServerURL), "/"); sURL != "" {
		return sURL
	}
	if envURL := strings.TrimRight(strings.TrimSpace(config.GetConfigWithDefault("SERVER_URL", "")), "/"); envURL != "" {
		return envURL
	}
	scheme := "http"
	if c.Secure() {
		scheme = "https"
	} else if proto := c.Get("X-Forwarded-Proto"); proto != "" {
		parts := strings.Split(proto, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			scheme = strings.TrimSpace(parts[0])
		}
	} else if c.Get("CF-Visitor") != "" && strings.Contains(c.Get("CF-Visitor"), `"scheme":"https"`) {
		scheme = "https"
	} else if strings.EqualFold(c.Get("X-Forwarded-Ssl"), "on") || strings.EqualFold(c.Get("X-Url-Scheme"), "https") || c.Protocol() == "https" {
		scheme = "https"
	}

	host := c.Hostname()
	if fHost := c.Get("X-Forwarded-Host"); fHost != "" {
		parts := strings.Split(fHost, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			host = strings.TrimSpace(parts[0])
		}
	} else if h := c.Get("Host"); h != "" {
		host = h
	}
	if host != "" {
		return scheme + "://" + host
	}
	return ""
}

type metaTags struct {
	Title        string
	Description  string
	OGType       string
	OGSiteName   string
	OGTitle      string
	OGDesc       string
	OGImage      string
	OGImageType  string
	OGURL        string
	TwitterCard  string
	Author       string
	ReleaseDate  string
	Tags         []string
}

var (
	titleTagRegex      = regexp.MustCompile(`(?i)<title>.*?</title>`)
	metaDescRegex      = regexp.MustCompile(`(?i)<meta\s+[^>]*name=["']description["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*name=["']description["'][^>]*>`)
	ogTypeRegex        = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:type["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*property=["']og:type["'][^>]*>`)
	ogSiteNameRegex    = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:site_name["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*property=["']og:site_name["'][^>]*>`)
	ogTitleRegex       = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:title["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*property=["']og:title["'][^>]*>`)
	ogDescRegex        = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:description["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*property=["']og:description["'][^>]*>`)
	ogImageRegex       = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:image["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*property=["']og:image["'][^>]*>`)
	ogImageSecureRegex = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:image:secure_url["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*property=["']og:image:secure_url["'][^>]*>`)
	ogImageTypeRegex   = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:image:type["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*property=["']og:image:type["'][^>]*>`)
	ogURLRegex         = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:url["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*property=["']og:url["'][^>]*>`)
	twitterCardRegex   = regexp.MustCompile(`(?i)<meta\s+[^>]*name=["']twitter:card["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*name=["']twitter:card["'][^>]*>`)
	twitterTitleRegex  = regexp.MustCompile(`(?i)<meta\s+[^>]*name=["']twitter:title["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*name=["']twitter:title["'][^>]*>`)
	twitterDescRegex   = regexp.MustCompile(`(?i)<meta\s+[^>]*name=["']twitter:description["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*name=["']twitter:description["'][^>]*>`)
	twitterImageRegex  = regexp.MustCompile(`(?i)<meta\s+[^>]*name=["']twitter:image["'][^>]*>|<meta\s+[^>]*content=["'][^"']*["'][^>]*name=["']twitter:image["'][^>]*>`)
	canonicalRegex     = regexp.MustCompile(`(?i)<link\s+[^>]*rel=["']canonical["'][^>]*>`)
)

func stripHTMLTags(s string) string {
	var builder strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			builder.WriteRune(' ')
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			builder.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func injectMetaTags(htmlDoc string, tags metaTags) string {
	htmlDoc = titleTagRegex.ReplaceAllString(htmlDoc, fmt.Sprintf("<title>%s</title>", html.EscapeString(tags.Title)))
	htmlDoc = metaDescRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta name="description" content="%s" />`, html.EscapeString(tags.Description)))
	htmlDoc = ogTypeRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta property="og:type" content="%s" />`, html.EscapeString(tags.OGType)))
	htmlDoc = ogSiteNameRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta property="og:site_name" content="%s" />`, html.EscapeString(tags.OGSiteName)))
	htmlDoc = ogTitleRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta property="og:title" content="%s" />`, html.EscapeString(tags.OGTitle)))
	htmlDoc = ogDescRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta property="og:description" content="%s" />`, html.EscapeString(tags.OGDesc)))
	htmlDoc = ogImageRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta property="og:image" content="%s" />`, html.EscapeString(tags.OGImage)))

	if tags.OGImage != "" {
		if ogImageSecureRegex.MatchString(htmlDoc) {
			htmlDoc = ogImageSecureRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta property="og:image:secure_url" content="%s" />`, html.EscapeString(tags.OGImage)))
		} else {
			htmlDoc = strings.Replace(htmlDoc, "</head>", fmt.Sprintf(`    <meta property="og:image:secure_url" content="%s" />`+"\n  </head>", html.EscapeString(tags.OGImage)), 1)
		}
		if tags.OGImageType != "" {
			if ogImageTypeRegex.MatchString(htmlDoc) {
				htmlDoc = ogImageTypeRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta property="og:image:type" content="%s" />`, html.EscapeString(tags.OGImageType)))
			} else {
				htmlDoc = strings.Replace(htmlDoc, "</head>", fmt.Sprintf(`    <meta property="og:image:type" content="%s" />`+"\n  </head>", html.EscapeString(tags.OGImageType)), 1)
			}
		}
	}

	if tags.OGURL != "" {
		if ogURLRegex.MatchString(htmlDoc) {
			htmlDoc = ogURLRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta property="og:url" content="%s" />`, html.EscapeString(tags.OGURL)))
		} else {
			htmlDoc = strings.Replace(htmlDoc, "</head>", fmt.Sprintf(`    <meta property="og:url" content="%s" />`+"\n  </head>", html.EscapeString(tags.OGURL)), 1)
		}
		if canonicalRegex.MatchString(htmlDoc) {
			htmlDoc = canonicalRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<link rel="canonical" href="%s" />`, html.EscapeString(tags.OGURL)))
		} else {
			htmlDoc = strings.Replace(htmlDoc, "</head>", fmt.Sprintf(`    <link rel="canonical" href="%s" />`+"\n  </head>", html.EscapeString(tags.OGURL)), 1)
		}
	}

	htmlDoc = twitterCardRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta name="twitter:card" content="%s" />`, html.EscapeString(tags.TwitterCard)))
	htmlDoc = twitterTitleRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta name="twitter:title" content="%s" />`, html.EscapeString(tags.OGTitle)))
	htmlDoc = twitterDescRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta name="twitter:description" content="%s" />`, html.EscapeString(tags.OGDesc)))
	htmlDoc = twitterImageRegex.ReplaceAllString(htmlDoc, fmt.Sprintf(`<meta name="twitter:image" content="%s" />`, html.EscapeString(tags.OGImage)))

	if tags.Author != "" {
		htmlDoc = strings.Replace(htmlDoc, "</head>", fmt.Sprintf(`    <meta name="author" content="%s" />`+"\n  </head>", html.EscapeString(tags.Author)), 1)
		if tags.OGType == "book" {
			htmlDoc = strings.Replace(htmlDoc, "</head>", fmt.Sprintf(`    <meta property="og:book:author" content="%s" />`+"\n  </head>", html.EscapeString(tags.Author)), 1)
		}
	}
	if tags.ReleaseDate != "" && tags.OGType == "book" {
		htmlDoc = strings.Replace(htmlDoc, "</head>", fmt.Sprintf(`    <meta property="og:book:release_date" content="%s" />`+"\n  </head>", html.EscapeString(tags.ReleaseDate)), 1)
	}
	for _, tag := range tags.Tags {
		if strings.TrimSpace(tag) != "" {
			htmlDoc = strings.Replace(htmlDoc, "</head>", fmt.Sprintf(`    <meta property="og:book:tag" content="%s" />`+"\n  </head>", html.EscapeString(strings.TrimSpace(tag))), 1)
		}
	}

	return htmlDoc
}

func serveEmbeddedFrontend(app *fiber.App, bookService services.BookService, settingsService services.SettingsService) {
	dist, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return
	}

	rawIndex, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return
	}

	serveIndex := func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		htmlDoc := string(rawIndex)

		// Determine site settings
		siteTitle := "NovelHub"
		siteDescription := "A modern, local light novel library and reader."
		logoURL := ""
		if pub, err := settingsService.Public(ctx); err == nil && pub != nil {
			if pub.Site.Title != "" {
				siteTitle = pub.Site.Title
			}
			if pub.Site.MetaDescription != "" {
				siteDescription = pub.Site.MetaDescription
			} else if pub.Site.Description != "" {
				siteDescription = pub.Site.Description
			}
			if pub.Site.Logo != "" {
				logoURL = pub.Site.Logo
			}
		}

		// Determine base origin URL
		serverURLSetting := ""
		if settingsService != nil {
			serverURLSetting = settingsService.ServerURL()
		}
		origin := resolveOrigin(c, serverURLSetting)

		path := c.Path()
		var (
			pageTitle       = siteTitle
			metaDescription = siteDescription
			ogType          = "website"
			ogTitle         = siteTitle
			ogDescription   = siteDescription
			ogImage         = ""
			ogImageType     = "image/png"
			ogURL           = ""
			twitterCard     = "summary"
			author          = ""
			releaseDate     = ""
			bookTags        []string
		)

		if origin != "" {
			ogURL = origin + c.OriginalURL()
		}

		// Determine site logo / image
		if logoURL != "" && !strings.HasSuffix(strings.ToLower(logoURL), ".svg") {
			if strings.HasPrefix(logoURL, "http://") || strings.HasPrefix(logoURL, "https://") {
				ogImage = logoURL
			} else if origin != "" {
				ogImage = origin + "/" + strings.TrimPrefix(logoURL, "/")
			}
			ext := strings.ToLower(filepath.Ext(ogImage))
			switch ext {
			case ".png":
				ogImageType = "image/png"
			case ".jpg", ".jpeg":
				ogImageType = "image/jpeg"
			case ".webp":
				ogImageType = "image/webp"
			case ".gif":
				ogImageType = "image/gif"
			}
		}
		if ogImage == "" {
			if origin != "" {
				ogImage = origin + "/pwa-512x512.png"
			} else {
				ogImage = "/pwa-512x512.png"
			}
			ogImageType = "image/png"
		}

		// Check if accessing a book route: /books/:id or /reader/:id
		var bookID string
		if strings.HasPrefix(path, "/books/") {
			bookID = strings.TrimPrefix(path, "/books/")
		} else if strings.HasPrefix(path, "/reader/") {
			bookID = strings.TrimPrefix(path, "/reader/")
		}
		if idx := strings.Index(bookID, "/"); idx != -1 {
			bookID = bookID[:idx]
		}
		bookID = strings.TrimSpace(bookID)

		if bookID != "" {
			book, err := bookService.GetBook(ctx, bookID)
			if err == nil && book != nil {
				ogType = "book"
				ogTitle = book.Title
				if book.AuthorName != nil && *book.AuthorName != "" {
					author = *book.AuthorName
					pageTitle = fmt.Sprintf("%s - %s | %s", book.Title, *book.AuthorName, siteTitle)
				} else {
					pageTitle = fmt.Sprintf("%s | %s", book.Title, siteTitle)
				}

				var metaParsed struct {
					Creator     string   `json:"creator"`
					Creators    []string `json:"creators"`
					Publisher   string   `json:"publisher"`
					Language    string   `json:"language"`
					Date        string   `json:"date"`
					Series      string   `json:"series"`
					SeriesIndex string   `json:"seriesIndex"`
					Subject     any      `json:"subject"`
				}
				if book.MetadataJSON != nil && *book.MetadataJSON != "" {
					_ = jsonx.UnmarshalString(*book.MetadataJSON, &metaParsed)
				}

				if author == "" {
					if metaParsed.Creator != "" {
						author = metaParsed.Creator
					} else if len(metaParsed.Creators) > 0 {
						author = strings.Join(metaParsed.Creators, ", ")
					}
				}

				if metaParsed.Date != "" {
					releaseDate = metaParsed.Date
				}

				if metaParsed.Subject != nil {
					switch sub := metaParsed.Subject.(type) {
					case []string:
						bookTags = append(bookTags, sub...)
					case []any:
						for _, item := range sub {
							if s, ok := item.(string); ok && s != "" {
								bookTags = append(bookTags, s)
							}
						}
					case string:
						for _, part := range strings.Split(sub, ",") {
							if trimmed := strings.TrimSpace(part); trimmed != "" {
								bookTags = append(bookTags, trimmed)
							}
						}
					}
				}

				if book.Description != nil && strings.TrimSpace(*book.Description) != "" {
					desc := stripHTMLTags(*book.Description)
					if len(desc) > 350 {
						desc = desc[:347] + "..."
					}
					metaDescription = desc
					ogDescription = desc
				} else {
					var details []string
					if author != "" {
						details = append(details, "Tác giả: "+author)
					}
					if metaParsed.Series != "" {
						seriesText := "Series: " + metaParsed.Series
						if metaParsed.SeriesIndex != "" {
							seriesText += " #" + metaParsed.SeriesIndex
						}
						details = append(details, seriesText)
					}
					if metaParsed.Publisher != "" {
						details = append(details, "NXB: "+metaParsed.Publisher)
					}
					if metaParsed.Language != "" {
						details = append(details, "Ngôn ngữ: "+strings.ToUpper(metaParsed.Language))
					}
					if len(bookTags) > 0 {
						limitTags := bookTags
						if len(limitTags) > 3 {
							limitTags = limitTags[:3]
						}
						details = append(details, "Thể loại: "+strings.Join(limitTags, ", "))
					}
					if len(details) > 0 {
						desc := strings.Join(details, " · ") + fmt.Sprintf(" | Đọc trên %s", siteTitle)
						metaDescription = desc
						ogDescription = desc
					} else {
						desc := fmt.Sprintf("Đọc \"%s\" trên %s.", book.Title, siteTitle)
						metaDescription = desc
						ogDescription = desc
					}
				}

				if book.CoverURL != nil && *book.CoverURL != "" {
					cover := *book.CoverURL
					if strings.HasPrefix(cover, "http://") || strings.HasPrefix(cover, "https://") {
						ogImage = cover
					} else if origin != "" {
						ogImage = origin + "/" + strings.TrimPrefix(cover, "/")
					}
					ext := strings.ToLower(filepath.Ext(ogImage))
					switch ext {
					case ".png":
						ogImageType = "image/png"
					case ".jpg", ".jpeg":
						ogImageType = "image/jpeg"
					case ".webp":
						ogImageType = "image/webp"
					case ".gif":
						ogImageType = "image/gif"
					}
					twitterCard = "summary"
				}
			}
		}

		htmlDoc = injectMetaTags(htmlDoc, metaTags{
			Title:       pageTitle,
			Description: metaDescription,
			OGType:      ogType,
			OGSiteName:  siteTitle,
			OGTitle:     ogTitle,
			OGDesc:      ogDescription,
			OGImage:     ogImage,
			OGImageType: ogImageType,
			OGURL:       ogURL,
			TwitterCard: twitterCard,
			Author:      author,
			ReleaseDate: releaseDate,
			Tags:        bookTags,
		})

		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.Status(fiber.StatusOK).SendString(htmlDoc)
	}

	app.Get("/", serveIndex)
	app.Get("/index.html", serveIndex)

	app.Use(static.New("", static.Config{
		FS:     dist,
		MaxAge: 31536000, // 1 year, correct only for the content-hashed assets/*
		// Caching sw.js freezes the old service worker forever; public/ files have no content hash.
		ModifyResponse: func(c fiber.Ctx) error {
			switch path := c.Path(); {
			case path == "/sw.js" || path == "/registerSW.js" || path == "/manifest.webmanifest":
				c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
			case strings.HasPrefix(path, "/locales/") || path == "/favicon.ico" ||
				path == "/logo.svg" || strings.HasPrefix(path, "/pwa-"):
				c.Set(fiber.HeaderCacheControl, "public, max-age=300, must-revalidate")
			}
			return nil
		},
		NotFoundHandler: func(c fiber.Ctx) error {
			if c.Method() != fiber.MethodGet {
				return c.Next()
			}
			path := c.Path()
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/locales/") || strings.HasPrefix(path, "/assets/") {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
			}
			return serveIndex(c)
		},
	}))
}
