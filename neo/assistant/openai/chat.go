package openai

import (
	"context"
	"fmt"
	"strings"

	chatMessage "github.com/yaoapp/yao/neo/message"
)

// Chat the chat struct
type Chat struct {
	ID       string `json:"chat_id"`
	ThreadID string `json:"thread_id"`
}

// NewChat create a new chat
func (ast *OpenAI) NewChat() {}

// Chat the chat
func (ast *OpenAI) Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) error {

	if ast.openai == nil {
		return fmt.Errorf("openai is not initialized")
	}

	requestMessages, err := ast.requestMessages(ctx, messages)
	if err != nil {
		return fmt.Errorf("request messages error: %s", err.Error())
	}

	_, ext := ast.openai.ChatCompletionsWith(ctx, requestMessages, option, cb)
	if ext != nil {
		return fmt.Errorf("openai chat completions with error: %s", ext.Message)
	}

	return nil
}

func (ast *OpenAI) requestMessages(ctx context.Context, messages []map[string]interface{}) ([]map[string]interface{}, error) {
	newMessages := []map[string]interface{}{}
	length := len(messages)
	for index, message := range messages {
		role, ok := message["role"].(string)
		if !ok {
			return nil, fmt.Errorf("role must be string")
		}

		content, ok := message["content"].(string)
		if !ok {
			return nil, fmt.Errorf("content must be string")
		}

		newMessage := map[string]interface{}{
			"role":    role,
			"content": content,
		}

		// Handle name if present
		if name, ok := message["name"].(string); ok {
			newMessage["name"] = name
		}

		newMessage["content"] = content

		// Special handling for user messages with JSON content last message
		if role == "user" && index == length-1 {
			content = strings.TrimSpace(content)
			msg, err := chatMessage.NewString(content)
			if err != nil {
				return nil, fmt.Errorf("new string error: %s", err.Error())
			}

			newMessage["content"] = msg.Text
			if msg.Attachments != nil {
				content, err := ast.withAttachments(ctx, msg)
				if err != nil {
					return nil, fmt.Errorf("with attachments error: %s", err.Error())
				}
				newMessage["content"] = content
			}
		}

		newMessages = append(newMessages, newMessage)
	}
	return newMessages, nil
}

func (ast *OpenAI) withAttachments(ctx context.Context, msg *chatMessage.Message) ([]map[string]interface{}, error) {
	contents := []map[string]interface{}{{"type": "text", "text": msg.Text}}
	images := []string{}
	for _, attachment := range msg.Attachments {
		if strings.HasPrefix(attachment.ContentType, "image/") {
			images = append(images, attachment.FileID)
		}
	}

	if len(images) == 0 {
		return contents, nil
	}

	for _, image := range images {
		bytes64, err := ast.ReadBase64(ctx, image)
		if err != nil {
			return nil, fmt.Errorf("read base64 error: %s", err.Error())
		}

		contents = append(contents, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]string{
				"url": fmt.Sprintf("data:image/jpeg;base64,%s", bytes64),
			},
		},
		)
	}

	return contents, nil
}
