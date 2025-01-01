package neo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/assistant/local"
	"github.com/yaoapp/yao/neo/assistant/openai"
	"github.com/yaoapp/yao/neo/message"
	"github.com/yaoapp/yao/neo/store"
	"github.com/yaoapp/yao/share"
)

// Lock the assistant list
var lock sync.Mutex = sync.Mutex{}

// Answer reply the message
func (neo *DSL) Answer(ctx Context, question string, c *gin.Context) error {
	messages, err := neo.chatMessages(ctx, question)
	if err != nil {
		msg := message.New().Error(err).Done()
		msg.Write(c.Writer)
		return err
	}

	// Get the assistant_id, chat_id
	res, err := neo.HookCreate(ctx, messages, c)
	if err != nil {
		msg := message.New().Error(err).Done()
		msg.Write(c.Writer)
		return err
	}

	// Select Assistant
	ast, err := neo.selectAssistant(res.AssistantID)
	if err != nil {
		return err
	}

	// Chat with AI
	return neo.chat(ast, ctx, messages, c)
}

// GetAssistants returns the list of assistants
func (neo *DSL) GetAssistants() []assistant.Assistant {
	return neo.AssistantList
}

// GetMentions returns the mention list
func (neo *DSL) GetMentions(keywords string) ([]Mention, error) {
	return neo.HookMention(context.Background(), keywords)
}

// GeneratePrompts generate prompts for the AI assistant
func (neo *DSL) GeneratePrompts(ctx Context, input string, c *gin.Context, silent ...bool) (string, error) {
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
func (neo *DSL) GenerateChatTitle(ctx Context, input string, c *gin.Context, silent ...bool) (string, error) {
	prompts := `
	Help me generate a title for the chat 
	1. The title should be a short and concise description of the chat.
	2. The title should be a single sentence.
	3. The title should be in same language as the chat.
	4. The title should be no more than 50 characters.
	`
	isSilent := false
	if len(silent) > 0 {
		isSilent = silent[0]
	}
	return neo.GenerateWithAI(ctx, input, "title", prompts, c, isSilent)
}

// GenerateWithAI generate content with AI, type can be "title", "prompts", etc.
func (neo *DSL) GenerateWithAI(ctx Context, input string, messageType string, systemPrompt string, c *gin.Context, silent bool) (string, error) {
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
	ast, err := neo.selectAssistant(res.AssistantID)
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
	content := []byte{}

	// Chat with AI in background
	go func() {
		err := ast.Chat(c.Request.Context(), messages, neo.Option, func(data []byte) int {
			select {
			case <-clientBreak:
				return 0 // break

			default:
				msg := message.NewOpenAI(data)
				if msg == nil {
					return 1 // continue
				}

				// Handle error
				if msg.Type == "error" {
					fail <- fmt.Errorf("%s", msg.Text)
					return 0 // break
				}

				// Append content and send message
				content = msg.Append(content)
				if !silent {
					value := msg.String()
					if value != "" {
						message.New().
							Map(map[string]interface{}{
								"text": value,
								"done": msg.IsDone,
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

		done <- true
	}()

	// Wait for completion or client disconnect
	select {
	case <-done:
		return string(content), nil
	case err := <-fail:
		return "", err
	case <-c.Writer.CloseNotify():
		clientBreak <- true
		return "", nil
	}
}

// Upload upload a file
func (neo *DSL) Upload(ctx Context, c *gin.Context) (*assistant.File, error) {
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
	ctx.Upload = &FileUpload{
		Bytes:       int(tmpfile.Size),
		Name:        tmpfile.Filename,
		ContentType: tmpfile.Header.Get("Content-Type"),
		Option:      option,
	}

	res, err := neo.HookCreate(ctx, []map[string]interface{}{}, c)
	if err != nil {
		return nil, err
	}

	// Select Assistant
	ast, err := neo.selectAssistant(res.AssistantID)
	if err != nil {
		return nil, err
	}

	return ast.Upload(ctx, tmpfile, reader, option)
}

// Download downloads a file
func (neo *DSL) Download(ctx Context, c *gin.Context) (*assistant.FileResponse, error) {
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
	ast, err := neo.selectAssistant(res.AssistantID)
	if err != nil {
		return nil, err
	}

	// Download file using the assistant
	return ast.Download(ctx.Context, fileID)
}

// chat chat with AI
func (neo *DSL) chat(ast assistant.API, ctx Context, messages []map[string]interface{}, c *gin.Context) error {
	if ast == nil {
		msg := message.New().Error("assistant is not initialized").Done()
		msg.Write(c.Writer)
		return fmt.Errorf("assistant is not initialized")
	}

	clientBreak := make(chan bool, 1)
	done := make(chan bool, 1)
	content := []byte{}

	// Chat with AI in background
	go func() {
		err := ast.Chat(c.Request.Context(), messages, neo.Option, func(data []byte) int {
			select {
			case <-clientBreak:
				return 0 // break

			default:
				msg := message.NewOpenAI(data)
				if msg == nil {
					return 1 // continue
				}

				// Handle error
				if msg.Type == "error" {
					value := msg.String()
					message.New().Error(value).Done().Write(c.Writer)
					return 0 // break
				}

				// Append content and send message
				content = msg.Append(content)
				value := msg.String()
				if value != "" {
					message.New().
						Map(map[string]interface{}{
							"text": value,
							"done": msg.IsDone,
						}).
						Write(c.Writer)
				}

				// Complete the stream
				if msg.IsDone {
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
			message.New().Error(err).Done().Write(c.Writer)
		}

		// Save chat history
		if len(content) > 0 {
			neo.saveHistory(ctx.Sid, ctx.ChatID, content, messages)
		}

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

// updateAssistantList update the assistant list
func (neo *DSL) updateAssistantList(list []assistant.Assistant) {
	lock.Lock()
	defer lock.Unlock()
	neo.AssistantList = list
	neo.AssistantMaps = make(map[string]assistant.Assistant)
	if list != nil {
		for _, assistant := range list {
			neo.AssistantMaps[assistant.ID] = assistant
		}
	}
}

// selectAssistant select the assistant
func (neo *DSL) selectAssistant(assistantID string) (assistant.API, error) {
	ast := neo.Assistant
	if assistantID != "" {
		ast, err := neo.newAssistant(assistantID)
		if err != nil {
			return nil, err
		}
		return ast, nil
	}
	return ast, nil
}

// newAssistant create a new assistant
func (neo *DSL) newAssistant(id string) (assistant.API, error) {
	// Try to find assistant in AssistantList first
	if id != "" && neo.AssistantMaps != nil {
		if ast, ok := neo.AssistantMaps[id]; ok {

			if ast.API != nil {
				return ast.API, nil
			}
			api, err := neo.newAssistantByConfig(&ast)
			if err != nil {
				return nil, err
			}
			ast.API = api
			return api, nil
		}
	}
	return neo.newAssistantByConnector(id)
}

// newAssistantByConfig create a new assistant from assistant configuration
func (neo *DSL) newAssistantByConfig(ast *assistant.Assistant) (assistant.API, error) {
	return neo.newAssistantByConnector(ast.Connector)
}

// newAssistantByConnector create a new assistant from connector id
func (neo *DSL) newAssistantByConnector(id string) (assistant.API, error) {
	// Moapi connector
	if id == "" || strings.HasPrefix(id, "moapi") {
		return neo.newMoapiAssistant(id)
	}

	// Other connector
	conn, err := connector.Select(id)
	if err != nil {
		return nil, fmt.Errorf("Neo assistant connector %s not support", id)
	}

	if conn.Is(connector.OPENAI) {
		api, err := openai.New(conn, id)
		if err != nil {
			return nil, fmt.Errorf("Create openai assistant error: %s", err.Error())
		}
		return api, nil
	}

	// Base on the assistant list hook
	api, err := local.New(conn, neo.Prompts, id)
	if err != nil {
		return nil, fmt.Errorf("Create local assistant error: %s", err.Error())
	}
	return api, nil
}

// newMoapiAssistant creates a new moapi assistant
func (neo *DSL) newMoapiAssistant(id string) (assistant.API, error) {
	model := "gpt-3.5-turbo"
	if strings.HasPrefix(id, "moapi:") {
		model = strings.TrimPrefix(id, "moapi:")
	}

	// Get the moapi setting
	url := share.MoapiHosts[0]
	if share.App.Moapi.Mirrors != nil {
		url = share.App.Moapi.Mirrors[0]
	}
	key := share.App.Moapi.Secret
	organization := share.App.Moapi.Organization

	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	// Check the moapi secret
	if key == "" {
		return nil, fmt.Errorf("The moapi secret is empty")
	}

	conn, err := connector.New(`moapi`, `__yao.moapi`, []byte(`{"name":"Moapi", "options":{"model": "`+model+`", "key": "`+key+`", "organization": "`+organization+`", "host": "`+url+`"}}`))
	if err != nil {
		return nil, fmt.Errorf("Create moapi assistant error: %s", err.Error())
	}

	api, err := openai.New(conn, strings.ReplaceAll(id, ":", "_"))
	if err != nil {
		return nil, fmt.Errorf("Create openai assistant error: %s", err.Error())
	}
	return api, nil
}

// createDefaultAssistant create a default assistant
func (neo *DSL) createDefaultAssistant() (assistant.API, error) {
	if neo.Use != "" {
		return neo.newAssistant(neo.Use)
	}
	return neo.newAssistant(neo.Connector)
}

// chatMessages get the chat messages
func (neo *DSL) chatMessages(ctx Context, content ...string) ([]map[string]interface{}, error) {

	history, err := neo.Store.GetHistory(ctx.Sid, ctx.ChatID)
	if err != nil {
		return nil, err
	}

	messages := []map[string]interface{}{}
	messages = append(messages, history...)
	if len(content) == 0 {
		return messages, nil
	}

	// Add user message
	messages = append(messages, map[string]interface{}{"role": "user", "content": content[0], "name": ctx.Sid})
	return messages, nil
}

// saveHistory save the history
func (neo *DSL) saveHistory(sid string, chatID string, content []byte, messages []map[string]interface{}) {

	if len(content) > 0 && sid != "" && len(messages) > 0 {
		err := neo.Store.SaveHistory(
			sid,
			[]map[string]interface{}{
				{"role": "user", "content": messages[len(messages)-1]["content"], "name": sid},
				{"role": "assistant", "content": string(content), "name": sid},
			},
			chatID,
			nil,
		)

		if err != nil {
			log.Error("Save history error: %s", err.Error())
		}
	}
}

// createStore create a new store
func (neo *DSL) createStore() error {

	var err error
	if neo.StoreSetting.Connector == "default" || neo.StoreSetting.Connector == "" {
		neo.Store, err = store.NewXun(neo.StoreSetting)
		return err
	}

	// other connector
	conn, err := connector.Select(neo.StoreSetting.Connector)
	if err != nil {
		return err
	}

	if conn.Is(connector.DATABASE) {
		neo.Store, err = store.NewXun(neo.StoreSetting)
		return err

	} else if conn.Is(connector.REDIS) {
		neo.Store = store.NewRedis()
		return nil

	} else if conn.Is(connector.MONGO) {
		neo.Store = store.NewMongo()
		return nil

	} else if conn.Is(connector.WEAVIATE) {
		neo.Store = store.NewWeaviate()
		return nil
	}

	return fmt.Errorf("%s store connector %s not support", neo.ID, neo.StoreSetting.Connector)
}

// sendMessage sends a message to the client
func (neo *DSL) sendMessage(w gin.ResponseWriter, data interface{}) error {
	if msg, ok := data.(map[string]interface{}); ok {
		if !message.New().Map(msg).Write(w) {
			return fmt.Errorf("failed to write message to stream")
		}
		return nil
	}
	return fmt.Errorf("invalid message data type")
}
