package assistant

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/process"
	chatctx "github.com/yaoapp/yao/neo/context"
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

// Execute implements the execute functionality
func (ast *Assistant) Execute(c *gin.Context, ctx chatctx.Context, input string, options map[string]interface{}) error {
	messages, err := ast.withHistory(ctx, input)
	if err != nil {
		return err
	}

	options = ast.withOptions(options)

	// Run init hook
	res, err := ast.HookInit(c, ctx, messages, options)
	if err != nil {
		chatMessage.New().
			Assistant(ast.ID, ast.Name, ast.Avatar).
			Error(err).
			Done().
			Write(c.Writer)
		return err
	}

	// Switch to the new assistant if necessary
	if res != nil && res.AssistantID != ctx.AssistantID {
		newAst, err := Get(res.AssistantID)
		if err != nil {
			chatMessage.New().
				Assistant(ast.ID, ast.Name, ast.Avatar).
				Error(err).
				Done().
				Write(c.Writer)
			return err
		}
		*ast = *newAst
	}

	// Handle next action
	if res != nil && res.Next != nil {
		return res.Next.Execute(c, ctx)
	}

	// Update options if provided
	if res != nil && res.Options != nil {
		options = res.Options
	}

	// messages
	if res != nil && res.Input != nil {
		messages = res.Input
	}

	// Only proceed with chat stream if no specific next action was handled
	return ast.handleChatStream(c, ctx, messages, options)
}

// Execute the next action
func (next *NextAction) Execute(c *gin.Context, ctx chatctx.Context) error {
	switch next.Action {

	case "process":
		if next.Payload == nil {
			return fmt.Errorf("payload is required")
		}

		name, ok := next.Payload["name"].(string)
		if !ok {
			return fmt.Errorf("process name should be string")
		}

		args := []interface{}{}
		if v, ok := next.Payload["args"].([]interface{}); ok {
			args = v
		}

		// Add context and writer to args
		args = append(args, ctx, c.Writer)
		p, err := process.Of(name, args...)
		if err != nil {
			return fmt.Errorf("get process error: %s", err.Error())
		}

		err = p.Execute()
		if err != nil {
			return fmt.Errorf("execute process error: %s", err.Error())
		}
		defer p.Release()

		return nil

	case "assistant":
		if next.Payload == nil {
			return fmt.Errorf("payload is required")
		}

		// Get assistant id
		id, ok := next.Payload["assistant_id"].(string)
		if !ok {
			return fmt.Errorf("assistant id should be string")
		}

		// Get assistant
		assistant, err := Get(id)
		if err != nil {
			return fmt.Errorf("get assistant error: %s", err.Error())
		}

		// Input
		input, ok := next.Payload["input"].(string)
		if !ok {
			return fmt.Errorf("input should be string")
		}

		// Options
		options := map[string]interface{}{}
		if v, ok := next.Payload["options"].(map[string]interface{}); ok {
			options = v
		}
		return assistant.Execute(c, ctx, input, options)

	case "exit":
		return nil

	default:
		return fmt.Errorf("unknown action: %s", next.Action)
	}
}

// handleChatStream manages the streaming chat interaction with the AI
func (ast *Assistant) handleChatStream(c *gin.Context, ctx chatctx.Context, messages []chatMessage.Message, options map[string]interface{}) error {
	clientBreak := make(chan bool, 1)
	done := make(chan bool, 1)
	contents := chatMessage.NewContents()

	// Chat with AI in background
	go func() {
		err := ast.streamChat(c, ctx, messages, options, clientBreak, done, contents)
		if err != nil {
			chatMessage.New().Error(err).Done().Write(c.Writer)
		}

		ast.saveChatHistory(ctx, messages, contents)
		done <- true
	}()

	// Wait for completion or client disconnect
	select {
	case <-done:
		return nil
	case <-c.Writer.CloseNotify():
		clientBreak <- true
		return nil
	}
}

// streamChat handles the streaming chat interaction
func (ast *Assistant) streamChat(
	c *gin.Context,
	ctx chatctx.Context,
	messages []chatMessage.Message,
	options map[string]interface{},
	clientBreak chan bool,
	done chan bool,
	contents *chatMessage.Contents) error {

	return ast.Chat(c.Request.Context(), messages, options, func(data []byte) int {
		select {
		case <-clientBreak:
			return 0 // break

		default:
			msg := chatMessage.NewOpenAI(data)
			if msg == nil {
				return 1 // continue
			}

			// Handle error
			if msg.Type == "error" {
				value := msg.String()
				res, hookErr := ast.HookFail(c, ctx, messages, contents.JSON(), fmt.Errorf("%s", value))
				if hookErr == nil && res != nil && (res.Output != "" || res.Error != "") {
					value = res.Output
					if res.Error != "" {
						value = res.Error
					}
				}
				chatMessage.New().Error(value).Done().Write(c.Writer)
				return 0 // break
			}

			// Append content and send message
			msg.AppendTo(contents)
			value := msg.String()
			if value != "" {
				// Handle stream
				res, err := ast.HookStream(c, ctx, messages, contents.Data)
				if err == nil && res != nil {

					if res.Next != nil {
						err = res.Next.Execute(c, ctx)
						if err != nil {
							chatMessage.New().Error(err.Error()).Done().Write(c.Writer)
						}

						done <- true
						return 0 // break
					}

					if res.Silent {
						return 1 // continue
					}
				}

				chatMessage.New().
					Map(map[string]interface{}{
						"assistant_id":     ast.ID,
						"assistant_name":   ast.Name,
						"assistant_avatar": ast.Avatar,
						"text":             value,
						"done":             msg.IsDone,
					}).
					Write(c.Writer)
			}

			// Complete the stream
			if msg.IsDone {
				// if value == "" {
				// 	msg.Write(c.Writer)
				// }

				res, hookErr := ast.HookDone(c, ctx, messages, contents.Data)
				if hookErr == nil && res != nil {
					if res.Output != nil {
						chatMessage.New().
							Map(map[string]interface{}{
								"text": res.Input,
								"done": true,
							}).
							Write(c.Writer)
					}

					if res.Next != nil {
						err := res.Next.Execute(c, ctx)
						if err != nil {
							chatMessage.New().Error(err.Error()).Done().Write(c.Writer)
						}
						done <- true
						return 0 // break
					}

				} else if value != "" {
					chatMessage.New().
						Map(map[string]interface{}{
							"assistant_id":     ast.ID,
							"assistant_name":   ast.Name,
							"assistant_avatar": ast.Avatar,
							"text":             value,
							"done":             true,
						}).
						Write(c.Writer)
				}

				done <- true
				return 0 // break
			}

			return 1 // continue
		}
	})
}

// saveChatHistory saves the chat history if storage is available
func (ast *Assistant) saveChatHistory(ctx chatctx.Context, messages []chatMessage.Message, contents *chatMessage.Contents) {
	if len(contents.Data) > 0 && ctx.Sid != "" && len(messages) > 0 {
		userMessage := messages[len(messages)-1]
		data := []map[string]interface{}{
			{
				"role":    "user",
				"content": userMessage.Content(),
				"name":    ctx.Sid,
			},
			{
				"role":             "assistant",
				"content":          contents.JSON(),
				"name":             ctx.Sid,
				"assistant_id":     ast.ID,
				"assistant_name":   ast.Name,
				"assistant_avatar": ast.Avatar,
			},
		}

		// Add mentions
		if userMessage.Mentions != nil {
			data[0]["mentions"] = userMessage.Mentions
		}

		storage.SaveHistory(ctx.Sid, data, ctx.ChatID, ctx.Map())
	}
}

func (ast *Assistant) withOptions(options map[string]interface{}) map[string]interface{} {
	if options == nil {
		options = map[string]interface{}{}
	}

	if ast.Options != nil {
		for key, value := range ast.Options {
			options[key] = value
		}
	}

	// Add functions
	if ast.Functions != nil {
		options["tools"] = ast.Functions
		if options["tool_choice"] == nil {
			options["tool_choice"] = "auto"
		}
	}

	return options
}

func (ast *Assistant) withPrompts(messages []chatMessage.Message) []chatMessage.Message {
	if ast.Prompts != nil {
		for _, prompt := range ast.Prompts {
			name := ast.Name
			if prompt.Name != "" {
				name = prompt.Name
			}
			messages = append(messages, *chatMessage.New().Map(map[string]interface{}{"role": prompt.Role, "content": prompt.Content, "name": name}))
		}
	}
	return messages
}

func (ast *Assistant) withHistory(ctx chatctx.Context, input string) ([]chatMessage.Message, error) {
	messages := []chatMessage.Message{}
	messages = ast.withPrompts(messages)
	if storage != nil {
		history, err := storage.GetHistory(ctx.Sid, ctx.ChatID)
		if err != nil {
			return nil, err
		}

		// Add history messages
		for _, h := range history {
			messages = append(messages, *chatMessage.New().Map(h))
		}
	}

	// Add user message
	messages = append(messages, *chatMessage.New().Map(map[string]interface{}{"role": "user", "content": input, "name": ctx.Sid}))
	return messages, nil
}

// Chat implements the chat functionality
func (ast *Assistant) Chat(ctx context.Context, messages []chatMessage.Message, option map[string]interface{}, cb func(data []byte) int) error {
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

func (ast *Assistant) requestMessages(ctx context.Context, messages []chatMessage.Message) ([]map[string]interface{}, error) {
	newMessages := []map[string]interface{}{}
	length := len(messages)

	for index, message := range messages {
		role := message.Role
		if role == "" {
			return nil, fmt.Errorf("role must be string")
		}

		content := message.Text
		if content == "" {
			return nil, fmt.Errorf("content must be string")
		}

		newMessage := map[string]interface{}{
			"role":    role,
			"content": content,
		}

		if name := message.Name; name != "" {
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
			if message.Attachments != nil {
				contents, err := ast.withAttachments(ctx, &message)
				if err != nil {
					return nil, fmt.Errorf("with attachments error: %s", err.Error())
				}

				// if current assistant is vision capable, add the contents directly
				if ast.vision {
					newMessage["content"] = contents
					continue
				}

				// If current assistant is not vision capable, add the description of the image
				if contents != nil {
					for _, content := range contents {
						newMessages = append(newMessages, content)
					}
				}
			}
		}

		newMessages = append(newMessages, newMessage)
	}
	return newMessages, nil
}

func (ast *Assistant) withAttachments(ctx context.Context, msg *chatMessage.Message) ([]map[string]interface{}, error) {
	contents := []map[string]interface{}{{"type": "text", "text": msg.Text}}
	if !ast.vision {
		contents = []map[string]interface{}{{"role": "user", "content": msg.Text}}
	}

	images := []string{}
	for _, attachment := range msg.Attachments {
		if strings.HasPrefix(attachment.ContentType, "image/") {
			if ast.vision {
				images = append(images, attachment.URL)
				continue
			}

			// If the current assistant is not vision capable, add the description of the image
			raw, err := jsoniter.MarshalToString(attachment)
			if err != nil {
				return nil, fmt.Errorf("marshal attachment error: %s", err.Error())
			}
			contents = append(contents, map[string]interface{}{
				"role":    "system",
				"content": raw,
			})
		}
	}

	if len(images) == 0 {
		return contents, nil
	}

	// If the current assistant is vision capable, add the image to the contents directly
	if ast.vision {
		for _, url := range images {

			// If the image is already a URL, add it directly
			if strings.HasPrefix(url, "http") {
				contents = append(contents, map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]string{
						"url": url,
					},
				})
				continue
			}

			// Read base64
			bytes64, err := ast.ReadBase64(ctx, url)
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

	// If the current assistant is not vision capable, add the description of the image

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
