package neo

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/neo/assistant"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
)

// Answer reply the message
func (neo *DSL) Answer(ctx chatctx.Context, question string, c *gin.Context) error {
	var err error
	var ast assistant.API = Neo.Assistant
	if ctx.AssistantID != "" {
		ast, err = neo.Select(ctx.AssistantID)
		if err != nil {
			return err
		}
	}
	_, err = ast.Execute(c, ctx, question, nil)
	return err
}

// Select select an assistant
func (neo *DSL) Select(id string) (assistant.API, error) {
	if id == "" {
		return Neo.Assistant, nil
	}
	return assistant.Get(id)
}

// GeneratePrompts generate prompts for the AI assistant
func (neo *DSL) GeneratePrompts(ctx chatctx.Context, input string, c *gin.Context, silent ...bool) (string, error) {
	prompts := `
	Optimize the prompts for the AI assistant
	1. Optimize prompts based on the user's input
	2. The prompts should be clear and specific
	3. The prompts should be in the same language as the input
	4. Keep the prompts concise but comprehensive
	5. DO NOT ASK USER FOR MORE INFORMATION, JUST GENERATE PROMPTS
	6. DO NOT ANSWER THE QUESTION, JUST GENERATE PROMPTS
	`
	isSilent := false
	if len(silent) > 0 {
		isSilent = silent[0]
	}
	return neo.GenerateWithAI(ctx, input, "prompts", prompts, c, isSilent)
}

// GenerateChatTitle generate the chat title
func (neo *DSL) GenerateChatTitle(ctx chatctx.Context, input string, c *gin.Context, silent ...bool) (string, error) {
	prompts := `
	Help me generate a title for the chat 
	1. The title should be a short and concise description of the chat.
	2. The title should be a single sentence.
	3. The title should be in same language as the chat.
	4. The title should be no more than 50 characters.
	5. ANSWER ONLY THE TITLE CONTENT, FOR EXAMPLE: Chat with AI is a valid title, but "Chat with AI" is not a valid title.
	`
	isSilent := false
	if len(silent) > 0 {
		isSilent = silent[0]
	}
	return neo.GenerateWithAI(ctx, input, "title", prompts, c, isSilent)
}

// GenerateWithAI generate content with AI, type can be "title", "prompts", etc.
func (neo *DSL) GenerateWithAI(ctx chatctx.Context, input string, messageType string, systemPrompt string, c *gin.Context, silent bool) (string, error) {
	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
		{
			"role":    "user",
			"content": input,
			"type":    messageType,
			"name":    ctx.Sid,
		},
	}

	res, err := neo.HookCreate(ctx, messages, c)
	if err != nil {
		return "", err
	}

	// Select Assistant
	ast, err := neo.Select(res.AssistantID)
	if err != nil {
		return "", err
	}

	if ast == nil {
		msg := message.New().Error("assistant is not initialized").Done()
		msg.Write(c.Writer)
		return "", fmt.Errorf("assistant is not initialized")
	}

	clientBreak := make(chan bool, 1)
	done := make(chan bool, 1)
	fail := make(chan error, 1)
	contents := message.NewContents()

	// Chat with AI in background
	go func() {
		msgList := []message.Message{}
		for _, vv := range messages {
			msg := message.New().Map(vv)
			if content, ok := vv["content"].(string); ok {
				msgs, err := message.NewContent(content)
				if err == nil {
					for _, v := range msgs {
						v.AssistantID = msg.AssistantID
						v.AssistantName = msg.AssistantName
						v.AssistantAvatar = msg.AssistantAvatar
						v.Role = msg.Role
						v.Name = msg.Name
						v.Mentions = msg.Mentions
						msgList = append(msgList, v)
					}
				}
			}
		}

		errorRaw := ""
		isFirstThink := true
		isThinking := false
		currentMessageID := ""
		err := ast.Chat(c.Request.Context(), msgList, neo.Option, func(data []byte) int {
			select {
			case <-clientBreak:
				return 0 // break

			default:
				msg := message.NewOpenAI(data, isThinking)
				if msg == nil {
					return 1 // continue
				}

				if msg.Pending {
					errorRaw += msg.Text
					return 1 // continue
				}

				// Handle error
				if msg.Type == "error" {
					fail <- fmt.Errorf("%s", msg.Text)
					return 0 // break
				}

				// for api reasoning_content response
				if msg.Type == "think" {
					if isFirstThink {
						msg.Text = "<think>\n" + msg.Text // add the think begin tag
						isFirstThink = false
						isThinking = true
					}
				}

				// for api reasoning_content response
				if isThinking && msg.Type != "think" {
					// add the think close tag
					end := message.New().Map(map[string]interface{}{"text": "\n</think>\n", "type": "think", "delta": true})
					end.Write(c.Writer)
					end.ID = currentMessageID
					end.AppendTo(contents)
					isThinking = false

					// Clear the token and make a new line
					contents.NewText([]byte{}, currentMessageID)
					contents.ClearToken()
				}

				// Append content and send message
				msg.AppendTo(contents)

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
					newMsg, err := message.NewString(tails, id)
					if err != nil {
						return
					}
					msgList = append(msgList, *newMsg)
				})

				if !silent {
					value := msg.String()
					if value != "" {
						message.New().
							Map(map[string]interface{}{
								"text":  value,
								"delta": true,
								"done":  msg.IsDone,
							}).
							Write(c.Writer)
					}
				}

				// Complete the stream
				if msg.IsDone {
					value := msg.String()
					if value == "" {
						msg.Write(c.Writer)
					}
					done <- true
					return 0 // break
				}

				return 1 // continue
			}
		})

		if err != nil {
			log.Error("Chat error: %s", err.Error())
			if !silent {
				message.New().Error(err).Done().Write(c.Writer)
			}
		}

		if errorRaw != "" {
			msg, err := message.NewStringError(errorRaw)
			if err != nil {
				log.Error("Error parsing error message: %s", err.Error())
			}
			msg.Write(c.Writer)
		}

		done <- true
	}()

	// Wait for completion or client disconnect
	select {
	case <-done:
		return contents.Text(), nil
	case err := <-fail:
		return "", err
	case <-c.Writer.CloseNotify():
		clientBreak <- true
		return "", nil
	}
}

// Upload upload a file
func (neo *DSL) Upload(ctx chatctx.Context, c *gin.Context) (*assistant.File, error) {
	// Get the file
	tmpfile, err := c.FormFile("file")
	if err != nil {
		return nil, err
	}

	reader, err := tmpfile.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		reader.Close()
		os.Remove(tmpfile.Filename)
	}()

	// Get option from form data option_xxx
	option := map[string]interface{}{}
	for key := range c.Request.Form {
		if strings.HasPrefix(key, "option_") {
			option[strings.TrimPrefix(key, "option_")] = c.PostForm(key)
		}
	}

	// Get file info
	ctx.Upload = &chatctx.FileUpload{
		Name:     tmpfile.Filename,
		Type:     tmpfile.Header.Get("Content-Type"),
		Size:     tmpfile.Size,
		TempFile: tmpfile.Filename,
	}

	// Default use the assistant in context
	ast := neo.Assistant
	if ctx.ChatID == "" {
		if ctx.AssistantID == "" {
			return nil, fmt.Errorf("assistant_id is required")
		}
		ast, err = neo.Select(ctx.AssistantID)
		if err != nil {
			return nil, err
		}
	}

	return ast.Upload(ctx, tmpfile, reader, option)
}

// Download downloads a file
func (neo *DSL) Download(ctx chatctx.Context, c *gin.Context) (*assistant.FileResponse, error) {
	// Get file_id from query string
	fileID := c.Query("file_id")
	if fileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	// Get assistant_id from context or query
	res, err := neo.HookCreate(ctx, []map[string]interface{}{}, c)
	if err != nil {
		return nil, err
	}

	// Select Assistant
	ast, err := neo.Select(res.AssistantID)
	if err != nil {
		return nil, err
	}

	// Download file using the assistant
	return ast.Download(ctx.Context, fileID)
}
