package telegram

import (
	"testing"
)

func TestBotConfig(t *testing.T) {
	cfg := &Config{
		BotToken:    "test-token",
		AllowedUsers: []int64{12345, 67890},
		Debug:       true,
	}

	if cfg.BotToken != "test-token" {
		t.Errorf("Expected token 'test-token', got %s", cfg.BotToken)
	}

	if len(cfg.AllowedUsers) != 2 {
		t.Errorf("Expected 2 allowed users, got %d", len(cfg.AllowedUsers))
	}

	if !cfg.Debug {
		t.Error("Expected debug to be true")
	}
}

func TestIsAllowed(t *testing.T) {
	// Test with allowlist
	cfg := &Config{
		BotToken:    "test",
		AllowedUsers: []int64{12345, 67890},
	}

	// Note: can't create Bot without valid token, so test the logic directly
	bot := &Bot{config: cfg}

	// Test allowed user
	if !bot.isAllowed(12345) {
		t.Error("Expected user 12345 to be allowed")
	}

	// Test blocked user
	if bot.isAllowed(99999) {
		t.Error("Expected user 99999 to be blocked")
	}

	// Test empty allowlist (allow all)
	cfg2 := &Config{
		BotToken:     "test",
		AllowedUsers: []int64{},
	}
	bot2 := &Bot{config: cfg2}

	if !bot2.isAllowed(12345) {
		t.Error("Expected any user to be allowed with empty allowlist")
	}
}

func TestMessageConversion(t *testing.T) {
	bot := &Bot{config: &Config{BotToken: "test"}}

	// Test nil message
	if msg := bot.convertMessage(nil); msg != nil {
		t.Error("Expected nil for nil message")
	}
}

func TestConversationManager(t *testing.T) {
	cm := NewConversationManager(10, 0) // 10 messages max, no TTL

	// Add messages
	cm.Add(123, "user", "Hello")
	cm.Add(123, "assistant", "Hi there!")

	// Get history
	history := cm.Get(123)
	if len(history) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(history))
	}

	// Check history
	if history[0].Role != "user" || history[0].Content != "Hello" {
		t.Error("First message mismatch")
	}

	if history[1].Role != "assistant" || history[1].Content != "Hi there!" {
		t.Error("Second message mismatch")
	}

	// Clear conversation
	cm.Clear(123)

	if cm.Get(123) != nil {
		t.Error("Expected nil after clear")
	}
}

func TestConversationManagerMaxHistory(t *testing.T) {
	cm := NewConversationManager(3, 0) // 3 messages max

	// Add more messages than max
	for i := 0; i < 5; i++ {
		cm.Add(123, "user", "msg")
	}

	history := cm.Get(123)
	if len(history) != 3 {
		t.Errorf("Expected 3 messages (max), got %d", len(history))
	}
}

func TestConversationManagerCount(t *testing.T) {
	cm := NewConversationManager(10, 0)

	// Add to different conversations
	cm.Add(123, "user", "Hello")
	cm.Add(456, "user", "Hi")
	cm.Add(789, "user", "Hey")

	if cm.Count() != 3 {
		t.Errorf("Expected 3 conversations, got %d", cm.Count())
	}
}

func TestButton(t *testing.T) {
	btn := Button{
		Text:         "Click Me",
		CallbackData: "click_123",
	}

	if btn.Text != "Click Me" {
		t.Errorf("Expected 'Click Me', got %s", btn.Text)
	}

	if btn.CallbackData != "click_123" {
		t.Errorf("Expected 'click_123', got %s", btn.CallbackData)
	}
}