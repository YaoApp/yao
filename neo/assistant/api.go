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
	"github.com/yaoapp/kun/utils"
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
	contents := chatMessage.NewContents()
	messages, err := ast.withHistory(ctx, input)
	if err != nil {
		return err
	}
	return ast.execute(c, ctx, messages, options, contents)
}

// Execute implements the execute functionality
func (ast *Assistant) execute(c *gin.Context, ctx chatctx.Context, input []chatMessage.Message, options map[string]interface{}, contents *chatMessage.Contents) error {

	if contents == nil {
		contents = chatMessage.NewContents()
	}
	options = ast.withOptions(options)

	// Add RAG and Version support
	ctx.RAG = rag != nil
	ctx.Version = ast.vision

	// Run init hook
	res, err := ast.HookInit(c, ctx, input, options, contents)
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
		return res.Next.Execute(c, ctx, contents)
	}

	// Update options if provided
	if res != nil && res.Options != nil {
		options = res.Options
	}

	// messages
	if res != nil && res.Input != nil {
		input = res.Input
	}

	// Only proceed with chat stream if no specific next action was handled
	return ast.handleChatStream(c, ctx, input, options, contents)
}

// Execute the next action
func (next *NextAction) Execute(c *gin.Context, ctx chatctx.Context, contents *chatMessage.Contents) error {
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
		input := chatMessage.Message{}
		_, has := next.Payload["input"]
		if !has {
			return fmt.Errorf("input is required")
		}

		switch v := next.Payload["input"].(type) {
		case string:
			messages := chatMessage.Message{}
			err := jsoniter.UnmarshalFromString(v, &messages)
			if err != nil {
				return fmt.Errorf("unmarshal input error: %s", err.Error())
			}
			input = messages

		case map[string]interface{}:
			msg, err := chatMessage.NewMap(v)
			if err != nil {
				return fmt.Errorf("unmarshal input error: %s", err.Error())
			}
			input = *msg

		case *chatMessage.Message:
			input = *v

		case chatMessage.Message:
			input = v

		default:
			return fmt.Errorf("input should be string or []chatMessage.Message")
		}

		// Options
		options := map[string]interface{}{}
		if v, ok := next.Payload["options"].(map[string]interface{}); ok {
			options = v
		}

		messages, err := assistant.withHistory(ctx, input)
		if err != nil {
			return fmt.Errorf("with history error: %s", err.Error())
		}

		fmt.Println("---messages ---")
		utils.Dump(messages)
		fmt.Println(`chatID: `, ctx.ChatID)

		return assistant.execute(c, ctx, messages, options, contents)

	case "exit":
		return nil

	default:
		return fmt.Errorf("unknown action: %s", next.Action)
	}
}

// GetPlaceholder returns the placeholder of the assistant
func (ast *Assistant) GetPlaceholder() *Placeholder {
	return ast.Placeholder
}

// Call implements the call functionality
func (ast *Assistant) Call(c *gin.Context, payload APIPayload) (interface{}, error) {
	scriptCtx, err := ast.Script.NewContext(payload.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer scriptCtx.Close()
	ctx := c.Request.Context()

	method := fmt.Sprintf("%sAPI", payload.Name)

	// Check if the method exists
	if !scriptCtx.Global().Has(method) {
		return nil, fmt.Errorf(HookErrorMethodNotFound)
	}

	return scriptCtx.CallWith(ctx, method, payload.Payload)
}

// handleChatStream manages the streaming chat interaction with the AI
func (ast *Assistant) handleChatStream(c *gin.Context, ctx chatctx.Context, messages []chatMessage.Message, options map[string]interface{}, contents *chatMessage.Contents) error {
	clientBreak := make(chan bool, 1)
	done := make(chan bool, 1)

	// Chat with AI in background
	go func() {
		err := ast.streamChat(c, ctx, messages, options, clientBreak, done, contents)
		if err != nil {
			chatMessage.New().Error(err).Done().Write(c.Writer)
		}

		ast.saveChatHistory(ctx, messages, contents)
		fmt.Printf("saveChatHistory %v\n", ctx.ChatID)
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

	errorRaw := ""
	isFirst := true
	currentMessageID := ""
	err := ast.Chat(c.Request.Context(), messages, options, func(data []byte) int {
		select {
		case <-clientBreak:
			return 0 // break

		default:
			msg := chatMessage.NewOpenAI(data)
			if msg == nil {
				return 1 // continue
			}

			if msg.Pending {
				errorRaw += msg.Text
				return 1 // continue
			}

			// Handle error
			if msg.Type == "error" {
				value := msg.String()
				res, hookErr := ast.HookFail(c, ctx, messages, fmt.Errorf("%s", value), contents)
				if hookErr == nil && res != nil && (res.Output != "" || res.Error != "") {
					value = res.Output
					if res.Error != "" {
						value = res.Error
					}
				}
				chatMessage.New().Error(value).Done().Write(c.Writer)
				return 0 // break
			}

			delta := msg.String()

			// Chunk the delta
			if delta != "" {

				msg.AppendTo(contents) // Append content and send message

				// Scan the tokens
				contents.ScanTokens(currentMessageID, func(token string, id string, begin bool, text string, tails string) {
					currentMessageID = id
					msg.ID = id
					msg.Type = token
					msg.Text = ""                                    // clear the text
					msg.Props = map[string]interface{}{"text": text} // Update props

					// End of the token clear the text
					if begin {
						return
					}

					// New message with the tails
					newMsg, err := chatMessage.NewString(tails, id)
					if err != nil {
						return
					}
					messages = append(messages, *newMsg)
				})

				// Handle stream
				res, err := ast.HookStream(c, ctx, messages, msg, contents)
				if err == nil && res != nil {

					if res.Next != nil {
						err = res.Next.Execute(c, ctx, contents)
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

				// Write the message to the client
				output := chatMessage.New().Map(map[string]interface{}{
					"text":  delta,
					"type":  msg.Type,
					"done":  msg.IsDone,
					"delta": true,
				})

				if isFirst {
					output.Assistant(ast.ID, ast.Name, ast.Avatar)
					isFirst = false
				}
				output.Write(c.Writer)
			}

			// Complete the stream
			if msg.IsDone {

				// if value == "" {
				// 	msg.Write(c.Writer)
				// }

				// Remove the last empty data
				contents.RemoveLastEmpty()

				res, hookErr := ast.HookDone(c, ctx, messages, contents)
				if hookErr == nil && res != nil {
					if res.Next != nil {
						err := res.Next.Execute(c, ctx, contents)
						if err != nil {
							chatMessage.New().Error(err.Error()).Done().Write(c.Writer)
						}

						done <- true
						return 0 // break
					}

				} else if delta != "" {
					chatMessage.New().
						Map(map[string]interface{}{
							"assistant_id":     ast.ID,
							"assistant_name":   ast.Name,
							"assistant_avatar": ast.Avatar,
							"text":             delta,
							"type":             "text",
							"delta":            true,
							"done":             true,
						}).
						Write(c.Writer)
				}

				// Hook execute error
				if hookErr != nil {
					chatMessage.New().Error(hookErr.Error()).Done().Write(c.Writer)
					done <- true
					return 0 // break
				}

				msg := chatMessage.New().Done()
				if res != nil && res.Output != nil {
					msg = chatMessage.New().
						Map(map[string]interface{}{
							"text": res.Input,
							"done": true,
						})
				}
				msg.Write(c.Writer)
				done <- true
				return 0 // break
			}

			return 1 // continue
		}
	})

	// Handle error
	if err != nil {
		return err
	}

	// raw error
	if errorRaw != "" {
		msg, err := chatMessage.NewStringError(errorRaw)
		if err != nil {
			return fmt.Errorf("error: %s", err.Error())
		}
		msg.Done().Write(c.Writer)
	}

	return nil
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

		// contents
		fmt.Println("---contents ---")
		if contents.Data != nil {
			fmt.Println("---contents.Data ---")
			for _, content := range contents.Data {
				fmt.Println(content.Map())
			}
			fmt.Println("---contents.Data end ---")
		}
		fmt.Println("---contents end ---")

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

	// Add Custom Options
	if ast.Options != nil {
		for key, value := range ast.Options {
			options[key] = value
		}
	}

	// Add tool_calls
	if ast.Tools != nil && ast.Tools.Tools != nil && len(ast.Tools.Tools) > 0 {
		if settings, has := connectorSettings[ast.Connector]; has && settings.Tools {
			options["tools"] = ast.Tools.Tools
			if options["tool_choice"] == nil {
				options["tool_choice"] = "auto"
			}
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

	// Add tool_calls
	if ast.Tools != nil && ast.Tools.Tools != nil && len(ast.Tools.Tools) > 0 {
		settings, has := connectorSettings[ast.Connector]
		if !has || !settings.Tools {
			raw, _ := jsoniter.MarshalToString(ast.Tools.Tools)
			messages = append(messages, *chatMessage.New().Map(map[string]interface{}{
				"role":    "system",
				"content": raw,
			}))

			// Add the default system prompts for tool calls
			messages = append(messages, *chatMessage.New().Map(map[string]interface{}{
				"role": "system",
				"content": "## Tool Calls Match Rules:\n" +
					"1. if the user's question is about the tool_calls, just answer one of the tool_calls, do not provide any additional information.\n" +
					"2. if the user's question is not about the tool_calls, just answer the user's question directly.\n" +
					"3. You can only use the functions defined in tool_calls. If none exist, reply directly to the user.",
			}))
			messages = append(messages, *chatMessage.New().Map(map[string]interface{}{
				"role": "system",
				"content": "## Tool Calls Response Rules:\n" +
					"1. The response should be a valid JSON object:\n" +
					"  1.1. e.g: <tool>{\"function\":\"function_name\",\"arguments\":{\"arg1\":\"xxxx\"}}</tool>\n" +
					"  1.2. strict the example format, do not add any additional information.\n" +
					"  1.3. The JSON object should be wrapped by <tool> and </tool>.\n" +
					"2. The structure of the JSON object is { \"arguments\": {...}, function:\"function_name\"}\n" +
					"3. The function_name should be the name of the function defined in tool_calls.\n" +
					"4. The arguments should be the arguments of the function defined in tool_calls.\n",
			}))

			// Add tool_calls prompts
			if ast.Tools.Prompts != nil && len(ast.Tools.Prompts) > 0 {
				for _, prompt := range ast.Tools.Prompts {
					messages = append(messages, *chatMessage.New().Map(map[string]interface{}{
						"role":    prompt.Role,
						"content": prompt.Content,
						"name":    prompt.Name,
					}))
				}
			}
		}
	}

	return messages
}

func (ast *Assistant) withHistory(ctx chatctx.Context, input interface{}) ([]chatMessage.Message, error) {

	var userMessage *chatMessage.Message = chatMessage.New()
	switch v := input.(type) {
	case string:
		userMessage.Map(map[string]interface{}{"role": "user", "content": v})
	case map[string]interface{}:
		userMessage.Map(v)
	case chatMessage.Message:
		userMessage = &v
	case *chatMessage.Message:
		userMessage = v
	default:
		return nil, fmt.Errorf("unknown input type: %T", input)
	}

	messages := []chatMessage.Message{}
	messages = ast.withPrompts(messages)
	if storage != nil {
		history, err := storage.GetHistory(ctx.Sid, ctx.ChatID)
		if err != nil {
			return nil, err
		}

		// Add history messages
		for _, h := range history {
			msgs, err := chatMessage.NewHistory(h)
			if err != nil {
				return nil, err
			}
			messages = append(messages, msgs...)
		}
	}

	// Add user message
	messages = append(messages, *userMessage)
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

		// Ignore the tool, think, error
		if message.Type == "tool" || message.Type == "think" || message.Type == "error" {
			continue
		}

		role := message.Role
		if role == "" {
			return nil, fmt.Errorf("role must be string")
		}

		content := message.String()
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
