package task

import (
	"context"
	"encoding/json"

	"github.com/yaoapp/xun/capsule"
)

// GetOriginalPrompt retrieves the first user message content from chat history.
// Used by retry (to re-execute with original prompt) and repeat (as fallback when instruction is empty).
// Returns interface{} — may be string (plain text) or []interface{} (multipart ContentPart[]).
func GetOriginalPrompt(_ context.Context, chatID string) interface{} {
	row, err := capsule.Global.Query().Table(tableMessage()).
		Select("props").
		Where("chat_id", "=", chatID).
		Where("role", "=", "user").
		OrderBy("sequence", "asc").
		First()
	if err != nil || row == nil {
		return ""
	}

	propsRaw, ok := row["props"]
	if !ok || propsRaw == nil {
		return ""
	}

	propsStr, ok := propsRaw.(string)
	if !ok {
		return ""
	}

	return extractContentFromProps(propsStr)
}

// GetOriginalPromptAsString retrieves the first user message as a plain string.
func GetOriginalPromptAsString(chatID string) string {
	content := GetOriginalPrompt(context.Background(), chatID)
	if content == nil {
		return ""
	}
	switch v := content.(type) {
	case string:
		return v
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

// GetFirstAssistantMessage retrieves the first assistant message content from chat history.
// Returns the text content as a string, or "" if not found.
func GetFirstAssistantMessage(chatID string) string {
	row, err := capsule.Global.Query().Table(tableMessage()).
		Select("props").
		Where("chat_id", "=", chatID).
		Where("role", "=", "assistant").
		OrderBy("sequence", "asc").
		First()
	if err != nil || row == nil {
		return ""
	}

	propsRaw, ok := row["props"]
	if !ok || propsRaw == nil {
		return ""
	}

	propsStr, ok := propsRaw.(string)
	if !ok {
		return ""
	}

	content := extractContentFromProps(propsStr)
	if content == nil {
		return ""
	}
	switch v := content.(type) {
	case string:
		return v
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

// extractContentFromProps parses message props JSON and returns the content field.
// Returns string for plain text, []interface{} for multipart, or "" on error.
func extractContentFromProps(propsJSON string) interface{} {
	var props map[string]interface{}
	if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
		return ""
	}

	content := props["content"]
	if content == nil {
		return ""
	}
	return content
}
