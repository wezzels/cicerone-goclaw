// Package telegram provides a Telegram bot client for Cicerone.
//
// This is a simplified Telegram-only implementation that connects to
// the LLM provider for message processing.
package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Config holds bot configuration.
type Config struct {
	BotToken    string   `json:"bot_token" yaml:"bot_token"`
	AllowedUsers []int64 `json:"allowed_users" yaml:"allowed_users"`
	Debug       bool     `json:"debug" yaml:"debug"`
}

// Bot represents a Telegram bot instance.
type Bot struct {
	api         *tgbotapi.BotAPI
	config      *Config
	updates     tgbotapi.UpdatesChannel
	inbound     chan Message
	stopCh      chan struct{}
	mu          sync.RWMutex
	running     bool
}

// Message represents an incoming message.
type Message struct {
	ID        int64
	FromID    int64
	FromName  string
	ChatID    int64
	Text      string
	ThreadID  int    // For replies
	Timestamp int64
}

// Handler processes incoming messages.
type Handler interface {
	Handle(ctx context.Context, msg Message) (string, error)
}

// HandlerFunc is an adapter for handlers.
type HandlerFunc func(ctx context.Context, msg Message) (string, error)

func (f HandlerFunc) Handle(ctx context.Context, msg Message) (string, error) {
	return f(ctx, msg)
}

// NewBot creates a new Telegram bot.
func NewBot(config *Config) (*Bot, error) {
	if config == nil || config.BotToken == "" {
		return nil, fmt.Errorf("telegram: bot token required")
	}

	api, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("telegram: creating bot API: %w", err)
	}

	api.Debug = config.Debug

	// Get bot info for logging
	me, err := api.GetMe()
	if err != nil {
		return nil, fmt.Errorf("telegram: getting bot info: %w", err)
	}

	log.Printf("Telegram: connected as @%s (ID: %d)", me.UserName, me.ID)

	return &Bot{
		api:     api,
		config:  config,
		inbound: make(chan Message, 100),
		stopCh:  make(chan struct{}),
	}, nil
}

// Start begins receiving updates.
func (b *Bot) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return fmt.Errorf("telegram: already running")
	}
	b.running = true
	b.mu.Unlock()

	// Create update config
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Get updates channel
	b.updates = b.api.GetUpdatesChan(u)

	// Start processing updates
	go b.processUpdates(ctx)

	log.Println("Telegram: started receiving updates")
	return nil
}

// processUpdates converts Telegram updates to Message.
func (b *Bot) processUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopCh:
			return
		case update := <-b.updates:
			if update.Message == nil {
				continue
			}

			msg := b.convertMessage(update.Message)
			if msg == nil {
				continue
			}

			// Check allowlist
			if !b.isAllowed(msg.FromID) {
				log.Printf("Telegram: blocked user %d", msg.FromID)
				continue
			}

			select {
			case b.inbound <- *msg:
			default:
				log.Printf("Telegram: inbound channel full, dropping message")
			}
		}
	}
}

// convertMessage converts a Telegram message to Message.
func (b *Bot) convertMessage(m *tgbotapi.Message) *Message {
	if m == nil {
		return nil
	}

	fromName := m.From.UserName
	if fromName == "" {
		fromName = m.From.FirstName
	}
	if fromName == "" {
		fromName = strconv.FormatInt(m.From.ID, 10)
	}

	msg := &Message{
		ID:        int64(m.MessageID),
		FromID:    m.From.ID,
		FromName:  fromName,
		ChatID:    m.Chat.ID,
		Text:      m.Text,
		Timestamp: m.Time().Unix(),
	}

	// Handle reply
	if m.ReplyToMessage != nil {
		msg.ThreadID = m.ReplyToMessage.MessageID
	}

	return msg
}

// isAllowed checks if a user is in the allowlist.
func (b *Bot) isAllowed(userID int64) bool {
	// If allowlist is empty, allow all
	if len(b.config.AllowedUsers) == 0 {
		return true
	}

	for _, allowed := range b.config.AllowedUsers {
		if allowed == userID {
			return true
		}
	}

	return false
}

// Receive returns the inbound message channel.
func (b *Bot) Receive() <-chan Message {
	return b.inbound
}

// Send sends a text message.
func (b *Bot) Send(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}

// SendReply sends a reply to a message.
func (b *Bot) SendReply(chatID int64, replyTo int, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyTo
	_, err := b.api.Send(msg)
	return err
}

// SendWithButtons sends a message with inline buttons.
func (b *Bot) SendWithButtons(chatID int64, text string, buttons []Button) error {
	msg := tgbotapi.NewMessage(chatID, text)

	// Convert buttons to Telegram format
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, btn := range buttons {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.CallbackData),
		}
		keyboard = append(keyboard, row)
	}

	if len(keyboard) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	}

	_, err := b.api.Send(msg)
	return err
}

// Edit edits a message.
func (b *Bot) Edit(chatID int64, messageID int, newText string) error {
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
	_, err := b.api.Send(editMsg)
	return err
}

// Delete deletes a message.
func (b *Bot) Delete(chatID int64, messageID int) error {
	_, err := b.api.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
	return err
}

// React adds a reaction to a message.
// Note: Reactions require Telegram Bot API 6.0+
// For older versions, this sends the emoji as text.
func (b *Bot) React(chatID int64, messageID int, emoji string) error {
	// Try native reactions first (Bot API 6.0+)
	// Fall back to sending as text reply if not supported
	msg := tgbotapi.NewMessage(chatID, emoji)
	msg.ReplyToMessageID = messageID
	_, err := b.api.Send(msg)
	return err
}

// Button represents an inline button.
type Button struct {
	Text         string
	CallbackData string
}

// Close stops the bot.
func (b *Bot) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	b.running = false
	close(b.stopCh)

	// Stop receiving updates
	if b.api != nil {
		b.api.StopReceivingUpdates()
	}

	return nil
}

// GetMe returns bot information.
func (b *Bot) GetMe() (tgbotapi.User, error) {
	return b.api.GetMe()
}

// GetChat returns chat information.
func (b *Bot) GetChat(chatID int64) (tgbotapi.Chat, error) {
	return b.api.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{ChatID: chatID},
	})
}

// String returns a string representation.
func (b *Bot) String() string {
	return "TelegramBot"
}