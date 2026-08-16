package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"novelhub/pkg/cache"
	"novelhub/pkg/config"
	"novelhub/pkg/database"
	"novelhub/pkg/logging"
)

func gracefulShutdown(server *FiberServer, done chan bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
	case <-server.Restart:
	}
	log.Info().Msg("shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.App.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	if server.Scheduler != nil {
		server.Scheduler.Stop()
	}
	if server.JobQueue != nil {
		queueCtx, queueCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer queueCancel()
		if err := server.JobQueue.StopContext(queueCtx); err != nil {
			log.Warn().Err(err).Msg("job queue did not stop before shutdown deadline")
		}
	}

	done <- true
}

func main() {
	if err := config.LoadEnv(); err != nil {
		log.Fatal().Err(err).Msg("failed to load env")
	}

	dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
	logWriter, err := logging.NewRotatingWriter(
		filepath.Join(dataDir, "logs", "novelhub.log"),
		int64(config.GetIntConfigWithDefault("LOG_MAX_SIZE_MB", 10))<<20,
		config.GetIntConfigWithDefault("LOG_MAX_FILES", 5),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize log writer")
	}
	defer logWriter.Close()
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	log.Logger = zerolog.New(zerolog.MultiLevelWriter(consoleWriter, logWriter)).With().Timestamp().Logger()

	if err := database.ApplyPendingRestore(); err != nil {
		log.Error().Err(err).Msg("pending restore was not applied; starting with current data")
	}

	if gcPercent := config.GetIntConfigWithDefault("GOGC", 200); gcPercent > 0 {
		debug.SetGCPercent(gcPercent)
	}

	db, err := database.NewSQLiteDB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect SQLite")
	}
	defer func() {
		if err := database.CheckpointWALAndClose(db); err != nil {
			log.Error().Err(err).Msg("failed to checkpoint and close database")
		}
	}()

	if err := database.ApplySchema(db); err != nil {
		log.Fatal().Err(err).Msg("failed to apply schema")
	}

	ramCache := cache.NewRamCache()

	server := NewHTTPServer()
	server.SetupServer(db, ramCache)

	host := config.GetConfigWithDefault("SERVER_HOST", "127.0.0.1")
	port := config.GetConfigWithDefault("SERVER_PORT", "3434")

	if config.GetBoolConfigWithDefault("ENABLE_PPROF", false) {
		go func() {
			// Debug endpoint on loopback only; opt-in via ENABLE_PPROF=1.
			if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
				log.Warn().Err(err).Msg("pprof server stopped")
			}
		}()
	}

	done := make(chan bool, 1)
	go gracefulShutdown(server, done)

	if err := server.App.Listen(
		fmt.Sprintf("%s:%s", host, port),
		fiber.ListenConfig{
			DisableStartupMessage: config.GetBoolConfigWithDefault("DISABLE_STARTUP_MESSAGE", false),
			EnablePrefork:         config.GetBoolConfigWithDefault("ENABLE_PREFORK", false),
		},
	); err != nil {
		log.Fatal().Err(err).Msg("failed to start API")
	}

	<-done
}
