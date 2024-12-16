package neo

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/assistant/base"
	"github.com/yaoapp/yao/neo/assistant/openai"
	"github.com/yaoapp/yao/neo/conversation"
	"github.com/yaoapp/yao/neo/message"
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
					message.New().Error(msg.Message.Text).Done().Write(c.Writer)
					return 0 // break
				}

				// Append content and send message
				content = msg.Append(content)
				if msg.Message != nil && msg.Message.Text != "" {
					message.New().
						Map(map[string]interface{}{
							"text": msg.Message.Text,
							"done": msg.Message.Done,
						}).
						Write(c.Writer)
				}

				// Complete the stream
				if msg.Message != nil && msg.Message.Done {
					if msg.Message.Text == "" {
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
	api, err := base.New(conn, neo.Prompts, id)
	if err != nil {
		return nil, fmt.Errorf("Create base assistant error: %s", err.Error())
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

	history, err := neo.Conversation.GetHistory(ctx.Sid, ctx.ChatID)
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
		err := neo.Conversation.SaveHistory(
			sid,
			[]map[string]interface{}{
				{"role": "user", "content": messages[len(messages)-1]["content"], "name": sid},
				{"role": "assistant", "content": string(content), "name": sid},
			},
			chatID,
		)

		if err != nil {
			log.Error("Save history error: %s", err.Error())
		}
	}
}

// createConversation create a new conversation
func (neo *DSL) createConversation() error {

	var err error
	if neo.ConversationSetting.Connector == "default" || neo.ConversationSetting.Connector == "" {
		neo.Conversation, err = conversation.NewXun(neo.ConversationSetting)
		return err
	}

	// other connector
	conn, err := connector.Select(neo.ConversationSetting.Connector)
	if err != nil {
		return err
	}

	if conn.Is(connector.DATABASE) {
		neo.Conversation, err = conversation.NewXun(neo.ConversationSetting)
		return err

	} else if conn.Is(connector.REDIS) {
		neo.Conversation = conversation.NewRedis()
		return nil

	} else if conn.Is(connector.MONGO) {
		neo.Conversation = conversation.NewMongo()
		return nil

	} else if conn.Is(connector.WEAVIATE) {
		neo.Conversation = conversation.NewWeaviate()
		return nil
	}

	return fmt.Errorf("%s conversation connector %s not support", neo.ID, neo.ConversationSetting.Connector)
}

// sendMessage sends a message to the client
func (neo *DSL) sendMessage(w gin.ResponseWriter, data interface{}) error {
	msg := message.New().Map(data.(map[string]interface{}))
	if !msg.Write(w) {
		return fmt.Errorf("failed to write message to stream")
	}
	return nil
}
