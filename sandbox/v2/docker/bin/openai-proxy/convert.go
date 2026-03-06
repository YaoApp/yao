package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Server) convertRequest(req *AnthropicRequest) *OpenAIRequest {
	maxTokens := req.MaxTokens
	if s.config.Options != nil {
		if mt, ok := s.config.Options["max_tokens"]; ok {
			switch v := mt.(type) {
			case float64:
				maxTokens = int(v)
			case int:
				maxTokens = v
			}
		}
	}

	temperature := req.Temperature
	if s.config.Options != nil {
		if temp, ok := s.config.Options["temperature"]; ok {
			if v, ok := temp.(float64); ok {
				temperature = &v
			}
		}
	}

	openaiReq := &OpenAIRequest{
		Model:       s.config.Model,
		MaxTokens:   maxTokens,
		Stream:      req.Stream,
		Temperature: temperature,
		TopP:        req.TopP,
		Stop:        req.StopSequences,
	}

	if s.config.Options != nil {
		openaiReq.ExtraOptions = make(map[string]interface{})
		for k, v := range s.config.Options {
			switch k {
			case "max_tokens", "temperature", "model", "key", "proxy":
				continue
			default:
				openaiReq.ExtraOptions[k] = v
			}
		}
	}

	openaiReq.Messages = s.convertMessages(req.Messages, req.System)

	if len(req.Tools) > 0 {
		openaiReq.Tools = s.convertTools(req.Tools)
	}

	if req.ToolChoice != nil {
		openaiReq.ToolChoice = s.convertToolChoice(req.ToolChoice)
	}

	return openaiReq
}

func (s *Server) convertMessages(msgs []AnthropicMsg, system interface{}) []OpenAIMsg {
	var result []OpenAIMsg

	if system != nil {
		systemText := extractSystemText(system)
		if systemText != "" {
			result = append(result, OpenAIMsg{
				Role:    "system",
				Content: systemText,
			})
		}
	}

	for _, msg := range msgs {
		converted := s.convertMessage(msg)
		result = append(result, converted...)
	}

	return result
}

func (s *Server) convertMessage(msg AnthropicMsg) []OpenAIMsg {
	var result []OpenAIMsg

	switch content := msg.Content.(type) {
	case string:
		result = append(result, OpenAIMsg{
			Role:    mapRole(msg.Role),
			Content: content,
		})

	case []interface{}:
		var toolResults []ContentBlock
		var otherContent []interface{}

		for _, item := range content {
			block := parseContentBlock(item)
			if block.Type == "tool_result" {
				toolResults = append(toolResults, block)
			} else {
				otherContent = append(otherContent, item)
			}
		}

		for _, tr := range toolResults {
			toolMsg := OpenAIMsg{
				Role:       "tool",
				ToolCallID: tr.ToolUseID,
				Content:    extractToolResultContent(tr.Content),
			}
			result = append(result, toolMsg)
		}

		if len(otherContent) > 0 {
			openaiContent := s.convertContentBlocks(otherContent)
			if len(openaiContent) == 1 && openaiContent[0].Type == "text" {
				result = append(result, OpenAIMsg{
					Role:    mapRole(msg.Role),
					Content: openaiContent[0].Text,
				})
			} else if len(openaiContent) > 0 {
				result = append(result, OpenAIMsg{
					Role:    mapRole(msg.Role),
					Content: openaiContent,
				})
			}
		}

		if msg.Role == "assistant" {
			toolCalls := extractToolUseBlocks(content)
			if len(toolCalls) > 0 {
				found := false
				for i := range result {
					if result[i].Role == "assistant" {
						result[i].ToolCalls = toolCalls
						found = true
						break
					}
				}
				if !found {
					result = append(result, OpenAIMsg{
						Role:      "assistant",
						Content:   "",
						ToolCalls: toolCalls,
					})
				}
			}
		}
	}

	return result
}

func (s *Server) convertContentBlocks(blocks []interface{}) []OpenAIContent {
	var result []OpenAIContent

	for _, item := range blocks {
		block := parseContentBlock(item)

		switch block.Type {
		case "text":
			result = append(result, OpenAIContent{
				Type: "text",
				Text: block.Text,
			})

		case "image":
			if block.Source != nil {
				imageURL := convertImageSource(block.Source)
				result = append(result, OpenAIContent{
					Type:     "image_url",
					ImageURL: imageURL,
				})
			}

		case "tool_use", "tool_result":
			continue
		}
	}

	return result
}

func convertImageSource(source *ImageSource) *OpenAIImageURL {
	if source == nil {
		return nil
	}

	switch source.Type {
	case "base64":
		mediaType := source.MediaType
		if mediaType == "" {
			mediaType = "image/jpeg"
		}
		return &OpenAIImageURL{
			URL: fmt.Sprintf("data:%s;base64,%s", mediaType, source.Data),
		}
	case "url":
		return &OpenAIImageURL{
			URL: source.URL,
		}
	}

	return nil
}

func (s *Server) convertTools(tools []AnthropicTool) []OpenAITool {
	var result []OpenAITool
	for _, tool := range tools {
		result = append(result, OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}
	return result
}

func (s *Server) convertToolChoice(choice *AnthropicToolChoice) interface{} {
	if choice == nil {
		return nil
	}
	switch choice.Type {
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "tool":
		return map[string]interface{}{
			"type": "function",
			"function": map[string]string{
				"name": choice.Name,
			},
		}
	case "none":
		return "none"
	}
	return "auto"
}

func (s *Server) convertResponse(resp *OpenAIResponse) *AnthropicResponse {
	result := &AnthropicResponse{
		ID:      generateID("msg_"),
		Type:    "message",
		Role:    "assistant",
		Content: []ContentBlock{},
		Model:   s.config.Model,
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]

		if content, ok := choice.Message.Content.(string); ok && content != "" {
			result.Content = append(result.Content, ContentBlock{
				Type: "text",
				Text: content,
			})
		}

		for _, tc := range choice.Message.ToolCalls {
			var input interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &input)

			result.Content = append(result.Content, ContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}

		stopReason := mapFinishReason(choice.FinishReason)
		result.StopReason = &stopReason
	}

	if resp.Usage != nil {
		result.Usage = &Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
	} else {
		result.Usage = &Usage{InputTokens: 0, OutputTokens: 0}
	}

	return result
}

func extractSystemText(system interface{}) string {
	switch s := system.(type) {
	case string:
		return s
	case []interface{}:
		var texts []string
		for _, item := range s {
			if block, ok := item.(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					if strings.HasPrefix(text, "x-anthropic-") {
						continue
					}
					texts = append(texts, text)
				}
			}
		}
		if len(texts) > 0 {
			return strings.Join(texts, "\n\n")
		}
	}
	return ""
}

func parseContentBlock(item interface{}) ContentBlock {
	var block ContentBlock
	switch v := item.(type) {
	case map[string]interface{}:
		if t, ok := v["type"].(string); ok {
			block.Type = t
		}
		if text, ok := v["text"].(string); ok {
			block.Text = text
		}
		if id, ok := v["id"].(string); ok {
			block.ID = id
		}
		if name, ok := v["name"].(string); ok {
			block.Name = name
		}
		if input, ok := v["input"]; ok {
			block.Input = input
		}
		if toolUseID, ok := v["tool_use_id"].(string); ok {
			block.ToolUseID = toolUseID
		}
		if content, ok := v["content"]; ok {
			block.Content = content
		}
		if isError, ok := v["is_error"].(bool); ok {
			block.IsError = isError
		}
		if source, ok := v["source"].(map[string]interface{}); ok {
			block.Source = parseImageSource(source)
		}
	}
	return block
}

func parseImageSource(source map[string]interface{}) *ImageSource {
	if source == nil {
		return nil
	}
	result := &ImageSource{}
	if t, ok := source["type"].(string); ok {
		result.Type = t
	}
	if mediaType, ok := source["media_type"].(string); ok {
		result.MediaType = mediaType
	}
	if data, ok := source["data"].(string); ok {
		result.Data = data
	}
	if url, ok := source["url"].(string); ok {
		result.URL = url
	}
	return result
}

func extractToolUseBlocks(content []interface{}) []OpenAIToolCall {
	var result []OpenAIToolCall
	for _, item := range content {
		block := parseContentBlock(item)
		if block.Type == "tool_use" {
			args, _ := json.Marshal(block.Input)
			result = append(result, OpenAIToolCall{
				ID:   block.ID,
				Type: "function",
				Function: OpenAIFunctionCall{
					Name:      block.Name,
					Arguments: string(args),
				},
			})
		}
	}
	return result
}

func extractToolResultContent(content interface{}) string {
	switch c := content.(type) {
	case string:
		return c
	case []interface{}:
		for _, item := range c {
			if block, ok := item.(map[string]interface{}); ok {
				if block["type"] == "text" {
					if text, ok := block["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

func mapRole(role string) string {
	switch role {
	case "user":
		return "user"
	case "assistant":
		return "assistant"
	default:
		return role
	}
}
