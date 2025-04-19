package assistant

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
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
func (ast *Assistant) Execute(c *gin.Context, ctx chatctx.Context, input interface{}, options map[string]interface{}, callback ...interface{}) (interface{}, error) {
	contents := chatMessage.NewContents()
	messages, err := ast.withHistory(ctx, input)
	if err != nil {
		return nil, err
	}
	return ast.execute(c, ctx, messages, options, contents, callback...)
}

// Execute implements the execute functionality
func (ast *Assistant) execute(c *gin.Context, ctx chatctx.Context, userInput interface{}, userOptions map[string]interface{}, contents *chatMessage.Contents, callback ...interface{}) (interface{}, error) {

	var input []chatMessage.Message

	switch v := userInput.(type) {
	case string:
		input = []chatMessage.Message{{Role: "user", Text: v}}

	case []interface{}:
		raw, err := jsoniter.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal input error: %s", err.Error())
		}
		err = jsoniter.Unmarshal(raw, &input)
		if err != nil {
			return nil, fmt.Errorf("unmarshal input error: %s", err.Error())
		}

	case []chatMessage.Message:
		input = v
	}

	if contents == nil {
		contents = chatMessage.NewContents()
	}
	options := ast.withOptions(userOptions)

	// Add RAG and Version support
	ctx.RAG = rag != nil
	ctx.Version = ast.vision

	// Run init hook
	res, err := ast.HookCreate(c, ctx, input, options, contents)
	if err != nil {
		chatMessage.New().
			Assistant(ast.ID, ast.Name, ast.Avatar).
			Error(err).
			Done().
			Write(c.Writer)
		return nil, err
	}

	// Update options if provided
	if res != nil && res.Options != nil {
		options = res.Options
	}

	// messages
	if res != nil && res.Input != nil {
		input = res.Input
	}

	// Handle next action
	// It's not used, return the new assistant_id and chat_id
	// if res != nil && res.Next != nil {
	// 	return res.Next.Execute(c, ctx, contents)
	// }

	// Switch to the new assistant if necessary
	if res != nil && res.AssistantID != "" && res.AssistantID != ctx.AssistantID {
		newAst, err := Get(res.AssistantID)
		if err != nil {
			chatMessage.New().
				Assistant(ast.ID, ast.Name, ast.Avatar).
				Error(err).
				Done().
				Write(c.Writer)
			return nil, err
		}

		// Reset Message Contents
		last := input[len(input)-1]
		input, err = newAst.withHistory(ctx, last)
		if err != nil {
			return nil, err
		}

		// Reset options
		options = newAst.withOptions(userOptions)

		// Update options if provided
		if res.Options != nil {
			options = res.Options
		}

		// Update assistant id
		ctx.AssistantID = res.AssistantID
		return newAst.handleChatStream(c, ctx, input, options, contents, callback...)
	}

	// Only proceed with chat stream if no specific next action was handled
	return ast.handleChatStream(c, ctx, input, options, contents, callback...)
}

// Execute the next action
func (next *NextAction) Execute(c *gin.Context, ctx chatctx.Context, contents *chatMessage.Contents, callback ...interface{}) (interface{}, error) {
	switch next.Action {

	// It's not used, because the process could be executed in the hook script
	// It may remove in the future
	// case "process":
	// 	if next.Payload == nil {
	// 		return fmt.Errorf("payload is required")
	// 	}

	// 	name, ok := next.Payload["name"].(string)
	// 	if !ok {
	// 		return fmt.Errorf("process name should be string")
	// 	}

	// 	args := []interface{}{}
	// 	if v, ok := next.Payload["args"].([]interface{}); ok {
	// 		args = v
	// 	}

	// 	// Add context and writer to args
	// 	args = append(args, ctx, c.Writer)
	// 	p, err := process.Of(name, args...)
	// 	if err != nil {
	// 		return fmt.Errorf("get process error: %s", err.Error())
	// 	}

	// 	err = p.Execute()
	// 	if err != nil {
	// 		return fmt.Errorf("execute process error: %s", err.Error())
	// 	}
	// 	defer p.Release()

	// 	return nil

	case "assistant":
		if next.Payload == nil {
			return nil, fmt.Errorf("payload is required")
		}

		// Get assistant id
		id, ok := next.Payload["assistant_id"].(string)
		if !ok {
			return nil, fmt.Errorf("assistant id should be string")
		}

		// Get assistant
		assistant, err := Get(id)
		if err != nil {
			return nil, fmt.Errorf("get assistant error: %s", err.Error())
		}

		// Input
		input := chatMessage.Message{}
		_, has := next.Payload["input"]
		if !has {
			return nil, fmt.Errorf("input is required")
		}

		// Retry mode
		retry := false
		_, has = next.Payload["retry"]
		if has {
			retry = next.Payload["retry"].(bool)
			ctx.Retry = retry
		}

		switch v := next.Payload["input"].(type) {
		case string:
			messages := chatMessage.Message{}
			err := jsoniter.UnmarshalFromString(v, &messages)
			if err != nil {
				return nil, fmt.Errorf("unmarshal input error: %s", err.Error())
			}
			input = messages

		case map[string]interface{}:
			msg, err := chatMessage.NewMap(v)
			if err != nil {
				return nil, fmt.Errorf("unmarshal input error: %s", err.Error())
			}
			input = *msg

		case *chatMessage.Message:
			input = *v

		case chatMessage.Message:
			input = v

		default:
			return nil, fmt.Errorf("input should be string or []chatMessage.Message")
		}

		// Options
		options := map[string]interface{}{}
		if v, ok := next.Payload["options"].(map[string]interface{}); ok {
			options = v
		}

		input.Hidden = true                    // not show in the history
		if input.Name == "" && ctx.Sid != "" { // add user id to the input
			input.Name = ctx.Sid
		}

		messages, err := assistant.withHistory(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("with history error: %s", err.Error())
		}

		// Send the progress message from application side instead
		// Create a new Text
		// Send loading message and mark as new
		// if !ctx.Silent {
		// 	msg := chatMessage.New().Map(map[string]interface{}{
		// 		"new":   true,
		// 		"role":  "assistant",
		// 		"type":  "loading",
		// 		"props": map[string]interface{}{"placeholder": "Calling " + assistant.Name},
		// 	})
		// 	msg.Assistant(assistant.ID, assistant.Name, assistant.Avatar)
		// 	msg.Write(c.Writer)
		// }
		newContents := chatMessage.NewContents()

		// Update the context id
		ctx.AssistantID = assistant.ID
		return assistant.execute(c, ctx, messages, options, newContents, callback...)

	case "exit":
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", next.Action)
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
		color.Red("Assistant Call: %s Method %s not found", ast.ID, method)
		return nil, fmt.Errorf(HookErrorMethodNotFound)
	}

	if payload.Args == nil || len(payload.Args) == 0 {
		return scriptCtx.CallWith(ctx, method)
	}

	return scriptCtx.CallWith(ctx, method, payload.Args...)
}

// handleChatStream manages the streaming chat interaction with the AI
func (ast *Assistant) handleChatStream(c *gin.Context, ctx chatctx.Context, messages []chatMessage.Message, options map[string]interface{}, contents *chatMessage.Contents, callback ...interface{}) (interface{}, error) {
	clientBreak := make(chan bool, 1)
	done := make(chan bool, 1)
	var result interface{} = nil
	var err error = nil

	requestCtx := c.Request.Context()
	go func() {
		var res interface{} = nil
		res, err = ast.streamChat(c, ctx, messages, options, clientBreak, contents, callback...)
		result = res
		done <- true
	}()

	// Wait for completion or client disconnect
	select {
	case <-done:
		if err != nil {
			return nil, err
		}
		return result, nil

	case <-requestCtx.Done():
		clientBreak <- true
		return nil, nil
	}
}

// streamChat handles the streaming chat interaction
func (ast *Assistant) streamChat(
	c *gin.Context,
	ctx chatctx.Context,
	messages []chatMessage.Message,
	options map[string]interface{},
	clientBreak chan bool,
	contents *chatMessage.Contents,
	callback ...interface{},
) (interface{}, error) {

	var cb interface{}
	if len(callback) > 0 {
		cb = callback[0]
	}

	errorRaw := ""
	isFirst := true
	isFirstThink := true
	isThinking := false

	toolsCount := 0
	currentMessageID := ""
	tokenID := ""
	beganAt := int64(0)
	var retry error = nil
	var result interface{} = nil // To save the result
	var content string = ""      // To save the content
	err := ast.Chat(c.Request.Context(), messages, options, func(data []byte) int {

		select {
		case <-clientBreak:
			return 0 // break

		default:
			msg := chatMessage.NewOpenAI(data, isThinking)
			if msg == nil {
				return 1 // continue
			}

			if msg.Pending {
				errorRaw += msg.Text
				return 1 // continue
			}

			// Retry mode
			msg.Retry = ctx.Retry   // Retry mode
			msg.Silent = ctx.Silent // Silent mode

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
				newMsg := chatMessage.New().Error(value).Done()
				newMsg.Retry = ctx.Retry
				newMsg.Silent = ctx.Silent
				newMsg.Callback(cb).Write(c.Writer)
				return 0 // break
			}

			// for api reasoning_content response
			if msg.Type == "think" {
				if isFirstThink {
					msg.Begin = time.Now().UnixNano()
					msg.Text = "<think>\n" + msg.Text // add the think begin tag
					isFirstThink = false
					isThinking = true
				}
			}

			// for api reasoning_content response
			if isThinking && msg.Type != "think" {
				// add the think close tag
				end := chatMessage.New().Map(map[string]interface{}{"text": "\n</think>\n", "type": "think", "delta": true})
				end.ID = currentMessageID
				end.Retry = ctx.Retry
				end.Silent = ctx.Silent
				end.End = time.Now().UnixNano()
				end.Begin = beganAt
				end.ToolID = tokenID

				end.Callback(cb).Write(c.Writer)
				end.AppendTo(contents)
				contents.UpdateType("think", map[string]interface{}{"text": contents.Text()}, chatMessage.Extra{ID: currentMessageID, End: time.Now().UnixNano()})
				isThinking = false

				// Clear the token and make a new line
				contents.NewText([]byte{}, chatMessage.Extra{ID: currentMessageID})

				// Clear the token
				contents.ClearToken(tokenID)
				beganAt = 0
				tokenID = ""
			}

			// for native tool_calls response, keep the first tool_calls_native message
			if msg.Type == "tool_calls_native" {

				if toolsCount > 1 {
					msg.Text = "" // clear the text
					msg.Type = "text"
					msg.IsNew = false
					return 1 // continue
				}

				if msg.IsBeginTool {

					if toolsCount == 1 {
						msg.IsNew = false
						msg.Text = "\n</tool>\n" // add the tool_calls close tag
					}

					if toolsCount == 0 {
						msg.Text = "\n<tool>\n" + msg.Text // add the tool_calls begin tag
					}

					toolsCount++
					msg.Begin = time.Now().UnixNano()
				}

				if msg.IsEndTool {
					msg.Text = msg.Text + "\n</tool>\n" // add the tool_calls close tag
					msg.End = time.Now().UnixNano()
				}
			}

			delta := msg.String()

			// Chunk the delta
			if delta != "" {

				msg.AppendTo(contents) // Append content

				// Scan the tokens
				contents.ScanTokens(currentMessageID, tokenID, beganAt, func(params message.ScanCallbackParams) {
					currentMessageID = params.MessageID
					msg.ID = params.MessageID
					msg.Type = params.Token
					msg.Text = ""                                                                 // clear the text
					msg.Props = map[string]interface{}{"text": params.Text, "id": params.TokenID} // Update props
					msg.Begin = params.BeganAt
					msg.End = params.EndAt
					msg.ToolID = params.TokenID

					// End of the token clear the text
					if params.Begin {
						tokenID = params.TokenID
						beganAt = params.BeganAt
						return
					}

					if params.End {
						tokenID = ""
						beganAt = 0
						return
					}

					// New message with the tails
					if params.Tails != "" {
						newMsg, err := chatMessage.NewString(params.Tails, params.MessageID)
						if err != nil {
							return
						}
						messages = append(messages, *newMsg)
					}
				})

				// Handle stream
				// The stream hook is not used, because there's no need to handle the stream output
				// if some thing need to be handled in future, we can use the stream hook again
				// ------------------------------------------------------------------------------
				// res, err := ast.HookStream(c, ctx, messages, msg, contents)
				// if err == nil && res != nil {

				// 	if res.Next != nil {
				// 		err = res.Next.Execute(c, ctx, contents)
				// 		if err != nil {
				// 			chatMessage.New().Error(err.Error()).Done().Write(c.Writer)
				// 		}

				// 		done <- true
				// 		return 0 // break
				// 	}

				// 	if res.Silent {
				// 		return 1 // continue
				// 	}
				// }
				// ------------------------------------------------------------------------------

				// Write the message to the stream
				msgType := msg.Type
				if msgType == "tool_calls_native" {
					msgType = "tool"
				}

				// Add the text content to the content
				if msgType == "text" || msgType == "" {
					content += msg.Text // Save the content
				}

				output := chatMessage.New().Map(map[string]interface{}{
					"text":  delta,
					"type":  msgType,
					"done":  msg.IsDone,
					"delta": true,
				})

				output.Retry = ctx.Retry   // Retry mode
				output.Silent = ctx.Silent // Silent mode
				if isFirst {
					output.Assistant(ast.ID, ast.Name, ast.Avatar)
					isFirst = false
				}

				if msg.Type == "think" || msg.Type == "tool" {
					output.Begin = msg.Begin
					output.End = msg.End
					output.ToolID = msg.ToolID
				}

				output.Callback(cb).Write(c.Writer)
			}

			// Complete the stream
			if msg.IsDone {

				// Send the last message to the client
				if delta != "" {
					chatMessage.New().
						Map(map[string]interface{}{
							"assistant_id":     ast.ID,
							"assistant_name":   ast.Name,
							"assistant_avatar": ast.Avatar,
							"text":             delta,
							"type":             "text",
							"delta":            true,
							"done":             true,
							"retry":            ctx.Retry,
							"silent":           ctx.Silent,
						}).
						Callback(cb).
						Write(c.Writer)
				}

				// Remove the last empty data
				contents.RemoveLastEmpty()
				res, hookErr := ast.HookDone(c, ctx, messages, contents)

				// Some error occurred in the hook, return the error
				if hookErr != nil {
					retry = hookErr
					return 0 // break
				}

				// Save the chat history
				ast.saveChatHistory(ctx, messages, contents)

				// If the hook is successful, execute the next action
				if res != nil && res.Next != nil {
					_, err := res.Next.Execute(c, ctx, contents, cb)
					if err != nil {
						chatMessage.New().Error(err.Error()).Done().Callback(cb).Write(c.Writer)
					}
					return 0 // break
				}

				// if the result is not nil, save the result
				if res != nil && res.Result != nil {
					result = res.Result
				}

				// The default output
				output := chatMessage.New().Done()
				if res != nil && res.Output != nil {
					output = chatMessage.New().Map(map[string]interface{}{"text": res.Output, "done": true})
					output.Retry = ctx.Retry
					output.Silent = ctx.Silent
				}

				// has result
				if res != nil && res.Result != nil && cb != nil {
					output.Result = res.Result // Add the result to the output  message
				}

				output.Callback(cb).Write(c.Writer)
				return 0 // break
			}

			return 1 // continue
		}
	})

	// retry
	if retry != nil {

		// Update the retry times
		ctx.RetryTimes = ctx.RetryTimes + 1 // Increment the retry times
		ctx.Retry = true                    // Set the retry mode

		// The maximum retry times is 9
		if ctx.RetryTimes > 9 {
			color.Red("Maximum retry times is 9, please check the error and fix it")
			// chatMessage.New().Error(retry.Error()).Done().Callback(cb).Write(c.Writer)
			return nil, retry
		}

		// Hook retry
		promptAny, retryErr := ast.HookRetry(c, ctx, messages, contents, exception.Trim(retry))
		if retryErr != nil {
			color.Red("%s, try to fix the error %d times, but failed with %s", exception.Trim(retry), ctx.RetryTimes, exception.Trim(retryErr))
			// chatMessage.New().Error(retry.Error()).Done().Callback(cb).Write(c.Writer)
			return nil, retry
		}

		if promptAny == nil {
			return nil, retry
		}

		var prompt string = ""
		switch v := promptAny.(type) {
		case NextAction:
			result, err := v.Execute(c, ctx, contents, cb)
			if err != nil {
				// chatMessage.New().Error(err.Error()).Done().Callback(cb).Write(c.Writer)
				return nil, retry
			}
			return result, nil

		case string:
			prompt = v
		}

		// Add the prompt to the messages
		retryMessages, retryErr := ast.retryMessages(messages, prompt)
		if retryErr != nil {
			color.Red("%s, try to fix the error %d times, but failed with %s", exception.Trim(retry), ctx.RetryTimes, exception.Trim(retryErr))
			// chatMessage.New().Error(retry.Error()).Done().Callback(cb).Write(c.Writer)
			return nil, retry
		}

		// Retry the chat
		retryContents := chatMessage.NewContents()
		return ast.execute(c, ctx, retryMessages, options, retryContents, cb)
	}

	// Handle error
	if err != nil {
		return nil, err
	}

	// raw error
	if errorRaw != "" {
		msg, err := chatMessage.NewStringError(errorRaw)
		if err != nil {
			return nil, fmt.Errorf("stream chat error %s", err.Error())
		}
		msg.Retry = ctx.Retry
		msg.Silent = ctx.Silent
		msg.Done().Callback(cb).Write(c.Writer)
	}

	// If the result is not nil, return the result
	if result != nil {
		return result, nil
	}

	// Return the content
	return strings.TrimSpace(content), nil
}

func (ast *Assistant) retryMessages(messages []chatMessage.Message, prompt string) ([]chatMessage.Message, error) {

	// Get the last user message
	var lastIndex int = -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			messages[i].Text = prompt
			lastIndex = i
			break
		}
	}

	if lastIndex == -1 {
		return nil, fmt.Errorf("no user message found")
	}

	// Remove the messages after the last user message
	messages = messages[:lastIndex+1]
	return messages, nil
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
				"name":             ast.ID,
				"assistant_id":     ast.ID,
				"assistant_name":   ast.Name,
				"assistant_avatar": ast.Avatar,
			},
		}

		// if the user message is hidden, just save the assistant message
		if userMessage.Hidden {
			data = []map[string]interface{}{data[1]}
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
			name := strings.ReplaceAll(ast.ID, ".", "_") // OpenAI only supports underscore in the name
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

			examples := []string{}
			for _, tool := range ast.Tools.Tools {
				example := tool.Example()
				examples = append(examples, example)
			}

			examplesStr := ""
			if len(examples) > 0 {
				examplesStr = "Examples:\n" + strings.Join(examples, "\n\n")
			}

			prompts := []map[string]interface{}{
				{
					"role":    "system",
					"name":    "TOOL_CALLS_SCHEMA",
					"content": raw,
				},
				{
					"role": "system",
					"name": "TOOL_CALLS_SCHEMA",
					"content": "## Tool Calls Schema Definition\n" +
						"Each tool call is defined with:\n" +
						"  - type: always 'function'\n" +
						"  - function:\n" +
						"    - name: function name\n" +
						"    - description: function description\n" +
						"    - parameters: function parameters with type and validation rules\n",
				},
				{
					"role": "system",
					"name": "TOOL_CALLS",
					"content": "## Tool Response Format\n" +
						"1. Only use tool calls when a function matches your task exactly\n" +
						"2. Each tool call must be wrapped in <tool> and </tool> tags\n" +
						"3. Tool call must be a valid JSON with:\n" +
						"   {\"function\": \"function_name\", \"arguments\": {parameters}}\n" +
						"4. Return the function's result as your response\n" +
						"5. One tool call per response\n" +
						"6. Arguments must match parameter types, rules and description\n\n" +
						examplesStr,
				},
				{
					"role": "system",
					"name": "TOOL_CALLS",
					"content": "## Tool Usage Guidelines\n" +
						"1. Use functions defined in TOOL_CALLS_SCHEMA only when they match your needs\n" +
						"2. If no matching function exists, respond normally as a helpful assistant\n" +
						"3. When using tools, arguments must match the schema definition exactly\n" +
						"4. All parameter values must strictly adhere to the validation rules specified in properties\n" +
						"5. Never skip or ignore any validation requirements defined in the schema",
				},
			}

			// Add tool_calls developer prompts
			if ast.Tools.Prompts != nil && len(ast.Tools.Prompts) > 0 {
				for _, prompt := range ast.Tools.Prompts {
					messages = append(messages, *chatMessage.New().Map(map[string]interface{}{
						"role":    prompt.Role,
						"content": prompt.Content,
						"name":    prompt.Name,
					}))
				}
			}

			// Add the prompts
			for _, prompt := range prompts {
				messages = append(messages, *chatMessage.New().Map(prompt))
			}

		}
	}

	return messages
}

func (ast *Assistant) withHistory(ctx chatctx.Context, input interface{}) ([]chatMessage.Message, error) {

	var userMessage *chatMessage.Message
	var inputMessages []*chatMessage.Message
	switch v := input.(type) {
	case string:
		userMessage = chatMessage.New().Map(map[string]interface{}{"role": "user", "content": v})

	case map[string]interface{}:
		userMessage = chatMessage.New().Map(v)

	case []interface{}:
		raw, err := jsoniter.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal input error: %s", err.Error())
		}
		err = jsoniter.Unmarshal(raw, &inputMessages)
		if err != nil {
			return nil, fmt.Errorf("unmarshal input error: %s", err.Error())
		}

	case chatMessage.Message:
		userMessage = &v
	case *chatMessage.Message:
		userMessage = v
	default:
		return nil, fmt.Errorf("unknown input type: %T", input)
	}

	messages := []chatMessage.Message{}
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

	// Add system prompts
	messages = ast.withPrompts(messages)

	// Add user message
	if userMessage != nil {
		messages = append(messages, *userMessage)
	}

	// Add input messages
	if len(inputMessages) > 0 {
		for _, msg := range inputMessages {
			if msg == nil || msg.Role == "" {
				continue
			}
			messages = append(messages, *msg)
		}
	}
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

// formatMessages processes messages to ensure they meet the required standards:
// 1. Filters out duplicate messages with identical content, role, and name
// 2. Moves system messages to the beginning while preserving the order of other messages
// 3. Ensures the first non-system message is a user message (removes leading assistant messages)
// 4. Ensures the last message is a user message (removes trailing assistant messages)
// 5. Merges consecutive assistant messages from the same assistant
func formatMessages(messages []map[string]interface{}) []map[string]interface{} {
	// Filter out duplicate messages with identical content, role, and name
	filteredMessages := []map[string]interface{}{
		{
			"role":    "system",
			"content": "Current time: " + time.Now().Format(time.RFC3339),
		},
	}
	seen := make(map[string]bool)

	for _, msg := range messages {
		// Create a unique key for each message based on role, content, and name
		role := msg["role"].(string)
		content := fmt.Sprintf("%v", msg["content"]) // Convert to string regardless of type

		// Get name if it exists
		name := ""
		if nameVal, exists := msg["name"]; exists {
			name = fmt.Sprintf("%v", nameVal)
		}

		// Create a unique key for this message
		key := fmt.Sprintf("%s:%s:%s", role, content, name)

		// If we haven't seen this message before, add it to filtered messages
		if !seen[key] {
			filteredMessages = append(filteredMessages, msg)
			seen[key] = true
		}
	}

	// Separate system messages while preserving the order of other messages
	systemMessages := []map[string]interface{}{}
	otherMessages := []map[string]interface{}{}

	for _, msg := range filteredMessages {
		if msg["role"].(string) == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// Ensure the first non-system message is a user message
	// If there are no user messages or the first message is not a user message, remove leading assistant messages
	validOtherMessages := []map[string]interface{}{}
	foundUserMessage := false

	for _, msg := range otherMessages {
		if msg["role"].(string) == "user" {
			foundUserMessage = true
			validOtherMessages = append(validOtherMessages, msg)
		} else if foundUserMessage {
			// Only keep assistant messages that come after a user message
			validOtherMessages = append(validOtherMessages, msg)
		}
		// Skip assistant messages that come before any user message
	}

	// If no valid messages remain, return just the system messages
	if len(validOtherMessages) == 0 {
		return systemMessages
	}

	// Ensure the last message is a user message
	// Remove any trailing assistant messages
	lastUserIndex := -1
	for i := len(validOtherMessages) - 1; i >= 0; i-- {
		if validOtherMessages[i]["role"].(string) == "user" {
			lastUserIndex = i
			break
		}
	}

	// If we found a user message, trim any assistant messages after it
	if lastUserIndex >= 0 && lastUserIndex < len(validOtherMessages)-1 {
		validOtherMessages = validOtherMessages[:lastUserIndex+1]
	}

	// If there are no user messages left after filtering, return just the system messages
	if len(validOtherMessages) == 0 {
		return systemMessages
	}

	// Combine system messages first, followed by other valid messages in their original order
	orderedMessages := append(systemMessages, validOtherMessages...)

	// Merge consecutive assistant messages
	mergedMessages := []map[string]interface{}{}
	var lastMessage map[string]interface{}

	for _, msg := range orderedMessages {
		// If this is the first message, just add it
		if lastMessage == nil {
			mergedMessages = append(mergedMessages, msg)
			lastMessage = msg
			continue
		}

		// If both current and last messages are from assistant, check if they can be merged
		if msg["role"].(string) == "assistant" && lastMessage["role"].(string) == "assistant" {
			// Get name information
			nameVal, hasName := msg["name"]

			// Prepare name prefix for the content
			namePrefix := ""
			if hasName {
				namePrefix = fmt.Sprintf("[%v]: ", nameVal)
			}

			// Merge the content, including name information if available
			lastContent := fmt.Sprintf("%v", lastMessage["content"])
			content := fmt.Sprintf("%v", msg["content"])

			// Add the name prefix to the content
			if namePrefix != "" {
				content = namePrefix + content
			}

			// Merge the messages
			lastMessage["content"] = lastContent + "\n" + content
			continue
		}

		// If we can't merge, add as a new message
		mergedMessages = append(mergedMessages, msg)
		lastMessage = msg
	}

	return mergedMessages
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
			if os.Getenv("YAO_AGENT_PRINT_REQUEST_MESSAGES") == "true" {
				raw, _ := jsoniter.MarshalToString(message)
				color.Red("Request Message Error, role is empty:")
				fmt.Println(raw)
			}
			return nil, fmt.Errorf("role must be string")
		}

		content := message.String()
		if content == "" {
			// fmt.Println("--------------------------------")
			// fmt.Println("Request Message Error")
			// utils.Dump(message)
			// fmt.Println("--------------------------------")
			// return nil, fmt.Errorf("content must be string")
			continue
		}

		newMessage := map[string]interface{}{
			"role":    role,
			"content": content,
		}

		// Keep the name for user messages
		if name := message.Name; name != "" {
			if role != "system" {
				newMessage["name"] = stringHash(name)
			} else {
				newMessage["name"] = name
			}
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

	// Process messages to standardize format, filter duplicates, and merge consecutive assistant messages
	processedMessages := formatMessages(newMessages)

	// For debug environment, print the request messages
	if os.Getenv("YAO_AGENT_PRINT_REQUEST_MESSAGES") == "true" {
		for _, message := range processedMessages {
			raw, _ := jsoniter.MarshalToString(message)
			log.Trace("[Request Message] %s", raw)
		}
	}

	return processedMessages, nil
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
