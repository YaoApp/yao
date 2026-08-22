package dsh

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 message construction for DSH SDK runtime.

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type initializeParams struct {
	CWD      string `json:"cwd"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	MaxToks  int    `json:"maxTokens,omitempty"`
}

type contentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	MediaType string `json:"mediaType,omitempty"`
	Data      string `json:"data,omitempty"`
	Name      string `json:"name,omitempty"`
}

type sessionPromptParams struct {
	SessionID     string         `json:"sessionId"`
	ContentBlocks []contentBlock `json:"contentBlocks"`
}

// buildInitializeMsg constructs the JSON-RPC initialize request.
func buildInitializeMsg(cwd, model string, maxTokens int) (string, error) {
	msg := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: initializeParams{
			CWD:      cwd,
			Provider: "deepseek-official",
			Model:    model,
			MaxToks:  maxTokens,
		},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal initialize: %w", err)
	}
	return string(b), nil
}

// buildSessionPromptMsg constructs the JSON-RPC session/prompt request.
func buildSessionPromptMsg(sessionID, userMessage string) (string, error) {
	msg := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "session/prompt",
		Params: sessionPromptParams{
			SessionID: sessionID,
			ContentBlocks: []contentBlock{
				{Type: "text", Text: userMessage},
			},
		},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal session/prompt: %w", err)
	}
	return string(b), nil
}

// buildSessionPromptMsgFromBlocks constructs a session/prompt with mixed content blocks.
func buildSessionPromptMsgFromBlocks(sessionID string, blocks []contentBlock) (string, error) {
	msg := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "session/prompt",
		Params: sessionPromptParams{
			SessionID:     sessionID,
			ContentBlocks: blocks,
		},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal session/prompt: %w", err)
	}
	return string(b), nil
}

// JSON-RPC response/notification parsing types.

type jsonRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// isResponse returns true if this is a JSON-RPC response (has id).
func (m *jsonRPCMessage) isResponse() bool {
	return m.ID != nil
}

// isNotification returns true if this is a JSON-RPC notification (has method, no id).
func (m *jsonRPCMessage) isNotification() bool {
	return m.ID == nil && m.Method != ""
}

// Session event envelope.

type sessionEventParams struct {
	SessionID string       `json:"sessionId"`
	Event     sessionEvent `json:"event"`
}

type sessionEvent struct {
	Type string          `json:"type"`
	Seq  int             `json:"seq"`
	Time int64           `json:"time"`
	Data json.RawMessage `json:"data"`
}

// Session status notification.

type sessionStatusParams struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}
