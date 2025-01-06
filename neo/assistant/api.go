package assistant

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/fs"
	chatMessage "github.com/yaoapp/yao/neo/message"
)

// Get get the assistant by id
func Get(id string) (*Assistant, error) {
	return LoadStore(id)
}

// GetByConnector get the assistant by connector
func GetByConnector(connector string, name string) (*Assistant, error) {
	id := "connector:" + connector

	assistant, exists := loaded.Get(id)
	if exists {
		return assistant, nil
	}

	data := map[string]interface{}{
		"assistant_id": id,
		"connector":    connector,
		"description":  "Default assistant for " + connector,
		"name":         name,
		"type":         "assistant",
	}

	assistant, err := loadMap(data)
	if err != nil {
		return nil, err
	}
	loaded.Put(assistant)
	return assistant, nil
}

// Chat implements the chat functionality
func (ast *Assistant) Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) error {
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

func (ast *Assistant) requestMessages(ctx context.Context, messages []map[string]interface{}) ([]map[string]interface{}, error) {
	newMessages := []map[string]interface{}{}

	// With Prompts
	if ast.Prompts != nil {
		for _, prompt := range ast.Prompts {
			message := map[string]interface{}{
				"role":    prompt.Role,
				"content": prompt.Content,
			}

			name := ast.Name
			if prompt.Name != "" {
				name = prompt.Name
			}

			message["name"] = name
			newMessages = append(newMessages, message)
		}
	}

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

		if name, ok := message["name"].(string); ok {
			newMessage["name"] = name
		}

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

func (ast *Assistant) withAttachments(ctx context.Context, msg *chatMessage.Message) ([]map[string]interface{}, error) {
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
		})
	}

	return contents, nil
}

// ReadBase64 implements base64 file reading functionality
func (ast *Assistant) ReadBase64(ctx context.Context, fileID string) (string, error) {
	data, err := fs.Get("data")
	if err != nil {
		return "", fmt.Errorf("get filesystem error: %s", err.Error())
	}

	exists, err := data.Exists(fileID)
	if err != nil {
		return "", fmt.Errorf("check file error: %s", err.Error())
	}
	if !exists {
		return "", fmt.Errorf("file %s not found", fileID)
	}

	content, err := data.ReadFile(fileID)
	if err != nil {
		return "", fmt.Errorf("read file error: %s", err.Error())
	}

	return base64.StdEncoding.EncodeToString(content), nil
}
