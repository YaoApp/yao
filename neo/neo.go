package neo

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/neo/command"
	"github.com/yaoapp/yao/neo/command/query"
	"github.com/yaoapp/yao/neo/conversation"
	"github.com/yaoapp/yao/neo/message"
	"github.com/yaoapp/yao/openai"
)

// API is a method on the Neo type
func (neo *DSL) API(router *gin.Engine, path string) error {

	// set the guard
	err := neo.setGuard(router)
	if err != nil {
		return err
	}

	// Cross-Domain
	neo.crossDomain(router, path)

	// api router chat
	router.GET(path, func(c *gin.Context) {

		sid := c.GetString("__sid")
		if sid == "" {
			sid = uuid.New().String()
		}

		content := c.Query("content")
		if content == "" {
			c.JSON(400, gin.H{"message": "content is required", "code": 400})
			return
		}

		// set the context
		ctx, cancel := command.NewContextWithCancel(sid, c.Query("context"))
		defer cancel()

		err = neo.Answer(ctx, content, c)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
		}

	})

	// api router chat history
	router.GET(path+"/history", func(c *gin.Context) {
		sid := c.GetString("__sid")
		if sid == "" {
			c.JSON(400, gin.H{"message": "sid is required", "code": 400})
			c.Done()
			return
		}

		history, err := neo.Conversation.GetHistory(sid)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
			return
		}

		c.JSON(200, history)
		c.Done()
	})

	// api router chat commands
	router.GET(path+"/commands", func(c *gin.Context) {

		commands, err := command.GetCommands()
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
			return
		}

		c.JSON(200, commands)
		c.Done()
	})

	// api router exit command mode
	router.POST(path, func(c *gin.Context) {
		sid := c.GetString("__sid")
		if sid == "" {
			c.JSON(400, gin.H{"message": "sid is required", "code": 400})
			c.Done()
			return
		}

		var payload map[string]interface{}
		err := c.ShouldBindJSON(&payload)
		if err != nil {
			c.JSON(400, gin.H{"message": err.Error(), "code": 400})
			c.Done()
			return
		}

		cmd, ok := payload["cmd"].(string)
		if !ok {
			c.JSON(400, gin.H{"message": "command is required", "code": 400})
			c.Done()
			return
		}

		switch cmd {
		case "ExitCommandMode":
			err := command.Exit(sid)
			if err != nil {
				c.JSON(500, gin.H{"message": err.Error(), "code": 500})
				c.Done()
				return
			}
			c.JSON(200, gin.H{"message": "success", "code": 200})
			c.Done()

		default:
			c.JSON(400, gin.H{"message": "command is not supported", "code": 400})
		}
	})

	return nil
}

// Answer the message
func (neo *DSL) Answer(ctx command.Context, question string, answer Answer) error {

	chanStream := make(chan *message.JSON, 1)
	chanError := make(chan error, 1)
	content := []byte{}

	// get the chat messages
	messages, err := neo.chatMessages(ctx, question)
	if err != nil {
		return err
	}

	// check the command
	cmd, isCommand := neo.matchCommand(ctx, messages)
	go func() {
		defer func() {
			close(chanStream)
			close(chanError)
		}()

		// execute the command
		if isCommand {

			req, err := cmd.NewRequest(ctx, neo.Conversation)
			if err != nil {
				chanError <- err
				return
			}

			err = req.Run(messages, func(msg *message.JSON) int {
				chanStream <- msg
				return 1
			})

			if err != nil {
				chanError <- err
			}

			return
		}

		// chat with AI
		_, ex := neo.AI.ChatCompletionsWith(ctx, messages, neo.Option, func(data []byte) int {
			chanStream <- message.NewOpenAI(data)
			return 1
		})

		if ex != nil {
			chanError <- fmt.Errorf("AI chat error: %s", ex.Message)
		}

		defer neo.saveHistory(ctx.Sid, content, messages)

	}()

	answer.Header("Content-Type", "text/event-stream;charset=utf-8")
	ok := answer.Stream(func(w io.Writer) bool {
		select {
		case err := <-chanError:
			if err != nil {
				message.New().Text(err.Error()).Write(w)
			}

			message.New().Done().Write(w)
			return false

		case msg := <-chanStream:
			if msg == nil {
				return true
			}
			msg.Write(w)
			content = msg.Append(content)
			return !msg.IsDone()

		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				message.New().Text(err.Error()).Write(w)
			}
			message.New().Done().Write(w)
			return false
		}
	})

	if !ok {
		answer.Status(500)
		return nil
	}

	answer.Status(200)
	return nil
}

// prompts get the prompts
func (neo *DSL) prompts() []map[string]interface{} {
	prompts := []map[string]interface{}{}
	for _, prompt := range neo.Prompts {
		message := map[string]interface{}{"role": prompt.Role, "content": prompt.Content}
		if prompt.Name != "" {
			message["name"] = prompt.Name
		}
		prompts = append(prompts, message)
	}

	return prompts
}

// prepare the messages
func (neo *DSL) prepare(ctx command.Context, messages []map[string]interface{}) []map[string]interface{} {
	if neo.Prepare == "" {
		return []map[string]interface{}{}
	}

	prompts := []map[string]interface{}{}
	p, err := process.Of(neo.Prepare, ctx, messages)
	if err != nil {
		color.Red("Neo prepare error: %s", err.Error())
		return prompts
	}

	data, err := p.Exec()
	if err != nil {
		color.Red("Neo prepare execute error: %s", err.Error())
		return prompts
	}

	items, ok := data.([]interface{})
	if !ok {
		color.Red("Neo prepare response is not array")
		return prompts
	}

	for i, item := range items {
		v, ok := item.(map[string]interface{})
		if !ok {
			color.Red("Neo prepare response [%d] is not map", i)
			continue
		}

		if _, ok := v["role"]; !ok {
			color.Red(`Neo prepare response [%d]["role"] required`, i)
			continue
		}

		if _, ok := v["content"]; !ok {
			color.Red(`Neo prepare response [%d]["content"] required`, i)
			continue
		}
		prompts = append(prompts, v)
	}

	return prompts
}

// chatMessages get the chat messages
func (neo *DSL) chatMessages(ctx command.Context, content string) ([]map[string]interface{}, error) {
	messages := append([]map[string]interface{}{}, neo.prompts()...)

	history, err := neo.Conversation.GetHistory(ctx.Sid)
	if err != nil {
		return nil, err
	}
	messages = append(messages, neo.prepare(ctx, messages)...) // Add prepare messages
	messages = append(messages, history...)
	messages = append(messages, map[string]interface{}{"role": "user", "content": content, "name": ctx.Sid})
	return messages, nil
}

// matchCommand match the command
func (neo *DSL) matchCommand(ctx command.Context, messages []map[string]interface{}) (*command.Command, bool) {
	if len(messages) < 1 {
		return nil, false
	}

	input, ok := messages[len(messages)-1]["content"].(string)
	if !ok {
		return nil, false
	}

	id, err := command.Match(ctx.Sid, query.Param{Stack: ctx.Stack, Path: ctx.Path}, input)
	if err == nil && id != "" {
		cmd, isCommand := command.Commands[id]
		return cmd, isCommand
	}

	return nil, false
}

// saveHistory save the history
func (neo *DSL) saveHistory(sid string, content []byte, messages []map[string]interface{}) {

	if len(content) > 0 && sid != "" && len(messages) > 0 {
		err := neo.Conversation.SaveHistory(
			sid,
			[]map[string]interface{}{
				{"role": "user", "content": messages[len(messages)-1]["content"], "name": sid},
				{"role": "assistant", "content": string(content), "name": sid},
			},
		)

		if err != nil {
			log.Error("Save history error: %s", err.Error())
		}
	}
}

func (neo *DSL) crossDomain(router *gin.Engine, path string) {

	if len(neo.Allows) == 0 {
		return
	}

	allowsMap := map[string]bool{}
	for _, allow := range neo.Allows {
		allow = strings.TrimPrefix(allow, "http://")
		allow = strings.TrimPrefix(allow, "https://")
		allowsMap[allow] = true
	}

	router.Use(func(c *gin.Context) {
		referer := c.Request.Referer()
		if referer != "" {

			if !api.IsAllowed(c, allowsMap) {
				c.JSON(403, gin.H{"message": referer + " not allowed", "code": 403})
				c.Abort()
				return
			}

			url, _ := url.Parse(referer)
			referer = fmt.Sprintf("%s://%s", url.Scheme, url.Host)
			c.Writer.Header().Set("Access-Control-Allow-Origin", referer)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
			c.Next()
		}
	})

	router.OPTIONS(path, func(c *gin.Context) { c.AbortWithStatus(204) })
}

func (neo *DSL) setGuard(router *gin.Engine) error {

	if neo.Guard == "" {
		router.Use(func(c *gin.Context) {
			token := strings.TrimSpace(strings.TrimPrefix(c.Query("token"), "Bearer "))
			if token == "" {
				c.JSON(403, gin.H{"message": "token is required", "code": 403})
				c.Abort()
				return
			}

			user := helper.JwtValidate(token)
			c.Set("__sid", user.SID)
			c.Next()
		})
		return nil
	}

	// validate the custom guard
	_, err := process.Of(neo.Guard)
	if err != nil {
		return err
	}

	// custom guard
	router.Use(api.ProcessGuard(neo.Guard))
	return nil
}

// NewAI create a new AI
func (neo *DSL) newAI() error {

	if neo.Connector == "" {
		return fmt.Errorf("%s connector is required", neo.ID)
	}

	conn, err := connector.Select(neo.Connector)
	if err != nil {
		return err
	}

	if conn.Is(connector.OPENAI) {
		ai, err := openai.New(neo.Connector)
		if err != nil {
			return err
		}
		neo.AI = ai
		return nil
	}

	return fmt.Errorf("%s connector %s not support, should be a openai", neo.ID, neo.Connector)
}

// newConversation create a new conversation
func (neo *DSL) newConversation() error {

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
