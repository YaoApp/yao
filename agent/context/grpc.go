package context

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// GRPCAgentInput holds the raw inputs from a gRPC AgentStream request.
type GRPCAgentInput struct {
	AssistantID string
	Messages    []byte
	Options     []byte
	AuthInfo    *types.AuthorizedInfo
	Cache       store.Store
	Writer      http.ResponseWriter
}

// GetGRPCAgentRequest parses a gRPC agent request and creates a Context + Options,
// mirroring openapi.go GetCompletionRequest.
//
// Flow: validate → parse messages → parse options → build Context → build Options → register interrupt
func GetGRPCAgentRequest(parent context.Context, input GRPCAgentInput) ([]Message, *Context, *Options, error) {
	if input.AssistantID == "" {
		return nil, nil, nil, fmt.Errorf("assistant_id is required")
	}

	messages, err := parseGRPCMessages(input.Messages)
	if err != nil {
		return nil, nil, nil, err
	}

	var rawOpts map[string]interface{}
	if len(input.Options) > 0 {
		if err := json.Unmarshal(input.Options, &rawOpts); err != nil {
			return nil, nil, nil, fmt.Errorf("invalid options JSON: %w", err)
		}
	}

	chatID := getChatIDFromOpts(rawOpts)
	ctx := New(parent, input.AuthInfo, chatID)

	ctx.Cache = input.Cache
	ctx.Writer = input.Writer
	ctx.AssistantID = input.AssistantID
	ctx.Locale = getStringOpt(rawOpts, "locale")
	ctx.Theme = getStringOpt(rawOpts, "theme")
	ctx.Referer = getRefererOpt(rawOpts)
	ctx.Accept = getAcceptOpt(rawOpts)
	ctx.Route = getStringOpt(rawOpts, "route")
	ctx.Metadata = getMapOpt(rawOpts, "metadata")
	ctx.Client = Client{Type: "grpc"}

	opts := &Options{
		Context: parent,
		Skip:    getSkipOpt(rawOpts),
		Mode:    getStringOpt(rawOpts, "mode"),
	}

	if connectorID := getStringOpt(rawOpts, "connector"); connectorID != "" {
		if _, err := connector.Select(connectorID); err == nil {
			opts.Connector = connectorID
		}
	}

	ctx.Interrupt = NewInterruptController()
	if err := Register(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to register context: %w", err)
	}
	ctx.Interrupt.Start(ctx.ID)

	return messages, ctx, opts, nil
}

func parseGRPCMessages(raw []byte) ([]Message, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("messages are required")
	}
	var messages []Message
	if err := json.Unmarshal(raw, &messages); err != nil {
		return nil, fmt.Errorf("invalid messages JSON: %w", err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages must not be empty")
	}
	return messages, nil
}

func getChatIDFromOpts(opts map[string]interface{}) string {
	if opts != nil {
		if v, ok := opts["chat_id"].(string); ok && v != "" {
			return v
		}
	}
	return GenChatID()
}

func getStringOpt(opts map[string]interface{}, key string) string {
	if opts == nil {
		return ""
	}
	v, _ := opts[key].(string)
	return v
}

func getRefererOpt(opts map[string]interface{}) string {
	r := getStringOpt(opts, "referer")
	if r != "" {
		return validateReferer(r)
	}
	return RefererAPI
}

func getAcceptOpt(opts map[string]interface{}) Accept {
	a := getStringOpt(opts, "accept")
	if a != "" {
		return validateAccept(a)
	}
	return AcceptStandard
}

func getMapOpt(opts map[string]interface{}, key string) map[string]interface{} {
	if opts == nil {
		return nil
	}
	v, _ := opts[key].(map[string]interface{})
	return v
}

func getSkipOpt(opts map[string]interface{}) *Skip {
	if opts == nil {
		return nil
	}
	raw, ok := opts["skip"]
	if !ok {
		return nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var skip Skip
	if err := json.Unmarshal(data, &skip); err != nil {
		return nil
	}
	return &skip
}
