package context

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/gou/store"
)

const (
	chatCachePrefix = "chat:messages:"
	chatCacheTTL    = time.Hour * 24 * 7 // 7 days
)

// filterNonAssistantMessages returns messages excluding assistant messages
func filterNonAssistantMessages(messages []Message) []Message {
	var filtered []Message
	for _, msg := range messages {
		if msg.Role != RoleAssistant {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// countUserMessages returns the number of user role messages
func countUserMessages(messages []Message) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == RoleUser {
			count++
		}
	}
	return count
}

// GetChatIDByMessages gets or generates a chat ID based on message content
// Matching strategy:
// - Only non-assistant messages (system, developer, user, tool) are used for matching
// - User adds a new message at the end each time
// - To detect continuation, we match messages BEFORE the last non-assistant message
// - If previous non-assistant messages match cached conversation → same chat
// - For first user message (even with system/developer messages): always generate new chat ID
func GetChatIDByMessages(cache store.Store, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("messages cannot be empty")
	}

	// Filter out assistant messages for matching
	nonAssistantMessages := filterNonAssistantMessages(messages)

	// Count user messages to determine matching strategy
	userMessageCount := countUserMessages(nonAssistantMessages)

	var chatID string
	var matched bool

	// Matching strategy based on user message count:
	// - 1 user message: generate new chat ID (cannot determine continuation)
	// - 2+ user messages: match all except last (which is the new user input)
	if userMessageCount >= 2 {
		// Match previous messages (all except last non-assistant message)
		matchMessages := nonAssistantMessages[:len(nonAssistantMessages)-1]
		hash, err := hashMessages(matchMessages)
		if err == nil {
			key := getKey(hash)
			if cachedID, ok := cache.Get(key); ok {
				if chatIDStr, ok := cachedID.(string); ok && chatIDStr != "" {
					chatID = chatIDStr
					matched = true
				}
			}
		}
	}

	// If no match, generate new chat ID
	if !matched {
		chatID = GenChatID()
	}

	// Cache the current messages for future matching
	// CacheChatID will handle filtering assistant messages
	// Next request will have one more message and will try to match current messages
	_ = CacheChatID(cache, messages, chatID)

	return chatID, nil
}

// CacheChatID cache the chat ID with all message prefixes for future matching
// It caches ALL prefixes of the message array to enable conversation continuation detection
// Assistant messages are automatically filtered out before caching
// Example: For messages [A,B,C], it caches hashes for [A], [A,B], and [A,B,C]
func CacheChatID(cache store.Store, messages []Message, chatID string) error {
	if len(messages) == 0 {
		return fmt.Errorf("messages cannot be empty")
	}

	if chatID == "" {
		return fmt.Errorf("chatID cannot be empty")
	}

	// Filter out assistant messages
	nonAssistantMessages := filterNonAssistantMessages(messages)
	if len(nonAssistantMessages) == 0 {
		return fmt.Errorf("no non-assistant messages to cache")
	}

	// Cache all prefixes of the non-assistant messages array
	// This allows detecting conversation continuation when new messages are added
	for length := 1; length <= len(nonAssistantMessages); length++ {
		prefix := nonAssistantMessages[:length]
		hash, err := hashMessages(prefix)
		if err != nil {
			continue // Skip this prefix if hashing fails
		}

		key := getKey(hash)
		// Ignore errors for individual cache sets
		_ = cache.Set(key, chatID, chatCacheTTL)
	}

	return nil
}

// GenChatID generate a new chat ID using NanoID algorithm
// safe: optional parameter, reserved for future safe mode implementation (collision detection)
func GenChatID(safe ...bool) string {
	// TODO: Implement safe mode with collision detection when needed
	// For now, NanoID provides sufficient uniqueness without collision checking

	// URL-safe alphabet (no ambiguous characters like 0/O, 1/l/I)
	const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
	const length = 16 // 16 characters provides good balance of uniqueness and readability

	id, err := gonanoid.Generate(alphabet, length)
	if err != nil {
		// Fallback to timestamp-based ID if NanoID generation fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	return id
}

// getKey generates a cache key for messages
func getKey(messageHash string) string {
	return chatCachePrefix + messageHash
}

// hashMessage generates a hash for a single message
func hashMessage(msg Message) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// hashMessages generates a combined hash for a slice of messages
// Note: Caller is responsible for filtering messages (e.g., removing assistant messages)
func hashMessages(messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("messages cannot be empty")
	}

	var hashes string
	for _, msg := range messages {
		hash, err := hashMessage(msg)
		if err != nil {
			return "", err
		}
		hashes += hash
	}

	// Generate final hash from combined hashes
	finalHash := sha256.Sum256([]byte(hashes))
	return hex.EncodeToString(finalHash[:]), nil
}
