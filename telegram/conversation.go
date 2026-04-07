package telegram

import (
	"sync"
	"time"
)

// ConversationManager manages conversation history per user/chat.
type ConversationManager struct {
	mu           sync.RWMutex
	conversations map[int64]*Conversation
	maxHistory   int // Max messages per conversation
	ttl          time.Duration
}

// Conversation holds chat history for a user.
type Conversation struct {
	ChatID    int64
	Messages  []MessageRecord
	LastSeen  time.Time
}

// MessageRecord stores a message in history.
type MessageRecord struct {
	Role      string    // "user" or "assistant"
	Content   string
	Timestamp time.Time
}

// NewConversationManager creates a new manager.
func NewConversationManager(maxHistory int, ttl time.Duration) *ConversationManager {
	return &ConversationManager{
		conversations: make(map[int64]*Conversation),
		maxHistory:   maxHistory,
		ttl:          ttl,
	}
}

// Add adds a message to the conversation.
func (cm *ConversationManager) Add(chatID int64, role, content string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, ok := cm.conversations[chatID]
	if !ok {
		conv = &Conversation{
			ChatID:   chatID,
			Messages: []MessageRecord{},
		}
		cm.conversations[chatID] = conv
	}

	conv.Messages = append(conv.Messages, MessageRecord{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	conv.LastSeen = time.Now()

	// Trim history
	if len(conv.Messages) > cm.maxHistory {
		conv.Messages = conv.Messages[len(conv.Messages)-cm.maxHistory:]
	}
}

// Get returns conversation history.
func (cm *ConversationManager) Get(chatID int64) []MessageRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conv, ok := cm.conversations[chatID]
	if !ok {
		return nil
	}

	// Return a copy
	history := make([]MessageRecord, len(conv.Messages))
	copy(history, conv.Messages)
	return history
}

// Clear clears a conversation.
func (cm *ConversationManager) Clear(chatID int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.conversations, chatID)
}

// Prune removes old conversations.
func (cm *ConversationManager) Prune() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	for chatID, conv := range cm.conversations {
		if now.Sub(conv.LastSeen) > cm.ttl {
			delete(cm.conversations, chatID)
		}
	}
}

// Count returns the number of active conversations.
func (cm *ConversationManager) Count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.conversations)
}