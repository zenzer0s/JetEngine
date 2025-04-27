package bot

import (
	"context"
	"fmt"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sirupsen/logrus"

	"jetengine/internal/config"
	"jetengine/internal/scraper"
	"jetengine/internal/storage"
)

// Handler holds dependencies for the Telegram bot handlers.
type Handler struct {
	bot     *tgbot.Bot
	cfg     config.Config
	repo    storage.Repository
	scraper scraper.Scraper
	log     logrus.FieldLogger
}

// NewHandler creates a new bot handler instance.
func NewHandler(cfg config.Config, repo storage.Repository, scraper scraper.Scraper, logger logrus.FieldLogger) (*Handler, error) {
	log := logger.WithField("component", "bot_handler")

	// Create the bot instance (without default handler for now)
	b, err := tgbot.New(cfg.TelegramBotToken)
	if err != nil {
		log.WithError(err).Error("Failed to create Telegram bot instance")
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	h := &Handler{
		bot:     b,
		cfg:     cfg,
		repo:    repo,
		scraper: scraper,
		log:     log,
	}

	// Register command handlers
	h.registerHandlers()

	// Register the default handler for text messages
	h.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "", tgbot.MatchTypeContains, h.defaultHandler)

	log.Info("Telegram bot handler initialized")
	return h, nil
}

// registerHandlers sets up the command and message handlers.
func (h *Handler) registerHandlers() {
	h.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, h.startHandler)
	h.log.Info("Registered /start command handler")
	// Add more handlers here later (e.g., /mylist)
}

// Start begins polling for updates from Telegram.
// This function blocks until the context is cancelled.
func (h *Handler) Start(ctx context.Context) {
	h.log.Info("Starting Telegram bot polling...")
	h.bot.Start(ctx) // Start polling
	h.log.Info("Telegram bot polling stopped.")
}

// startHandler handles the /start command.
func (h *Handler) startHandler(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	log := h.log.WithFields(logrus.Fields{
		"user_id": userID,
		"command": "/start",
	})
	log.Info("Received /start command")

	// Send a welcome message
	welcomeMessage := "Welcome to JetEngine! Send me a website link, and I'll save its metadata for you."
	_, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   welcomeMessage,
	})

	if err != nil {
		log.WithError(err).Error("Failed to send welcome message")
	}
}

func (h *Handler) defaultHandler(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if h.repo == nil || h.scraper == nil || h.log == nil {
		// Log or handle the error gracefully
		fmt.Println("Handler dependencies are not initialized")
		return
	}
	// For now, just log that we received a message
	// In Step 6, this will parse URLs, call the scraper, and save to repo.
	h.log.WithFields(logrus.Fields{
		"user_id": update.Message.From.ID,
		"text":    update.Message.Text,
	}).Debug("Received unhandled message (default handler)")

	// Optionally, send a placeholder response
	// _, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
	// 	ChatID: update.Message.Chat.ID,
	// 	Text:   "Send me a URL to save, or use /start or /mylist.",
	// })
}

// TODO: Implement callbackHandler for inline buttons in Step 7
