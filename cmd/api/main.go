package main

import (
	"context"
	"fmt"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"novelhub/pkg/cache"
	"novelhub/pkg/config"
	"novelhub/pkg/database"
)

func gracefulShutdown(server *FiberServer, done chan bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Info().Msg("shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.App.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	if server.JobQueue != nil {
		server.JobQueue.Stop()
	}

	done <- true
}

func main() {
	if err := config.LoadEnv(); err != nil {
		log.Fatal().Err(err).Msg("failed to load env")
	}

	if gcPercent := config.GetIntConfigWithDefault("GOGC", 200); gcPercent > 0 {
		debug.SetGCPercent(gcPercent)
	}

	db, err := database.NewSQLiteDB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect SQLite")
	}
	defer db.Close()

	if err := database.ApplySchema(db, "db/schema"); err != nil {
		log.Fatal().Err(err).Msg("failed to apply schema")
	}

	if err := database.SeedSuperAdmin(db); err != nil {
		log.Fatal().Err(err).Msg("failed to seed admin user")
	}

	ramCache := cache.NewRamCache()

	server := NewHTTPServer()
	server.SetupServer(db, ramCache)

	host := config.GetConfigWithDefault("SERVER_HOST", "127.0.0.1")
	port := config.GetConfigWithDefault("SERVER_PORT", "3434")

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
