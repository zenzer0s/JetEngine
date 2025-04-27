package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"jetengine/internal/bot"
	"jetengine/internal/config"
	"jetengine/internal/scraper"
	"jetengine/internal/storage"
)

func main() {
	// --- Configuration Loading ---
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// --- Logger Setup ---
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// TODO: Make log level configurable via cfg.LogLevel
	log.SetLevel(logrus.InfoLevel)

	log.WithFields(logrus.Fields{
		"badgerdb_path": cfg.BadgerDBPath,
	}).Info("Configuration loaded successfully")

	// --- Initialize Components ---
	log.Info("Initializing components...")

	// Database
	repo, err := storage.NewBadgerRepository(cfg.BadgerDBPath, log)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		log.Info("Closing database...")
		if err := repo.Close(); err != nil {
			log.WithError(err).Error("Error closing database")
		}
	}()

	// Scraper
	scraperService := scraper.NewRodScraper(log)
	// TODO: Add scraperService.Close() if needed and call in defer

	// Bot Handler
	botHandler, err := bot.NewHandler(cfg, repo, scraperService, log)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram bot handler: %v", err)
	}

	// --- Application Startup ---
	log.Info("Starting JetEngine...")

	// Create context that listens for interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop() // Ensure stop is called to release resources

	// Start the bot polling in a separate goroutine
	go botHandler.Start(ctx)

	log.Info("JetEngine is running. Press Ctrl+C to exit.")

	// --- Wait for Shutdown Signal ---
	<-ctx.Done() // Block here until the context is cancelled (Ctrl+C)

	// --- Graceful Shutdown ---
	log.Info("Shutting down JetEngine...")
	stop() // Explicitly call stop to ensure signal handling is cleaned up

	// The deferred repo.Close() will run now.
	// Add cleanup for other components if needed (e.g., scraper).

	log.Info("JetEngine shut down gracefully.")
}
