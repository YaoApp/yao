package neo

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/neo/conversation"
	"github.com/yaoapp/yao/neo/message"
	"github.com/yaoapp/yao/openai"
)

// Answer reply the message
func (neo *DSL) Answer(ctx Context, question string, c *gin.Context) error {
	// get the chat messages
	messages, err := neo.chatMessages(ctx, question)
	if err != nil {
		return err
	}

	clientBreak := make(chan bool, 1)
	done := make(chan bool, 1)
	content := []byte{}

	// Execute the command or chat with AI in the background
	go func() {

		// chat with AI
		c.Header("Content-Type", "text/event-stream;charset=utf-8")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		_, ex := neo.AI.ChatCompletionsWith(ctx, messages, neo.Option, func(data []byte) int {

			select {
			case <-clientBreak:
				return 0 // break
			default:

				msg := message.NewOpenAI(data)
				if msg == nil {
					return 1 // continue success
				}

				if msg.Error != "" {
					neo.send(ctx, msg, messages, content, c)
					return 0 // break
				}

				content = msg.Append(content)
				err := neo.send(ctx, msg, messages, content, c)
				if err != nil {
					c.Status(500)
					return 0 // break
				}

				// Complete the stream
				if msg.IsDone() {
					done <- true
					return 0 // break
				}

				return 1 // continue success
			}
		})

		// Throw the error
		if ex != nil {
			log.Error("Neo chat error: %s", ex.Message)
			c.Status(200)
			done <- true
			return
		}

		// save the history
		neo.saveHistory(ctx.Sid, ctx.ChatID, content, messages)
		c.Status(200)

		// Complete the stream
		done <- true

	}()

	select {
	case <-done:
		return nil
	case <-c.Writer.CloseNotify():
		clientBreak <- true
		return nil
	}

}

// Send send the message to the stream
func (neo *DSL) send(ctx Context, msg *message.JSON, messages []map[string]interface{}, content []byte, c *gin.Context) error {

	w := c.Writer

	if msg.Error != "" {
		msg.Write(w)
		return nil
	}

	// Directly write the message
	if neo.Write == "" {
		ok := msg.Write(c.Writer)
		if !ok {
			return fmt.Errorf("Stream write error")
		}
		return nil
	}

	// Execute the custom write hook get the response
	args := []interface{}{ctx, messages, msg, string(content), w}
	p, err := process.Of(neo.Write, args...)
	if err != nil {
		msg.Write(w)
		color.Red("Neo custom write error: %s", err.Error())
		return fmt.Errorf("Stream write error: %s", err.Error())
	}

	err = p.WithSID(ctx.Sid).Execute()
	if err != nil {
		log.Error("Neo custom write error: %s", err.Error())
		msg.Write(w)
		return nil
	}
	defer p.Release()

	res := p.Value()
	if res == nil {
		color.Red("Neo custom write return null")
		return fmt.Errorf("Neo custom write return null")
	}

	// Send the custom write response to the stream
	if messages, ok := res.([]interface{}); ok {
		for _, new := range messages {
			if v, ok := new.(map[string]interface{}); ok {
				newMsg := message.New().Map(v)
				newMsg.Write(w)
			}
		}
		return nil
	}

	color.Red("Neo custom write should return an array of response")
	return fmt.Errorf("Neo should return an array of response")
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
func (neo *DSL) prepare(ctx Context, messages []map[string]interface{}) []map[string]interface{} {
	if neo.Prepare == "" {
		return []map[string]interface{}{}
	}

	prompts := []map[string]interface{}{}
	p, err := process.Of(neo.Prepare, ctx, messages)
	if err != nil {
		color.Red("Neo prepare error: %s", err.Error())
		return prompts
	}

	err = p.WithSID(ctx.Sid).Execute()
	if err != nil {
		color.Red("Neo prepare execute error: %s", err.Error())
		return prompts
	}
	defer p.Release()

	data := p.Value()
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
func (neo *DSL) chatMessages(ctx Context, content string) ([]map[string]interface{}, error) {

	history, err := neo.Conversation.GetHistory(ctx.Sid, ctx.ChatID)
	if err != nil {
		return nil, err
	}
	messages := append([]map[string]interface{}{}, neo.prompts()...)
	messages = append(messages, history...)
	messages = append(messages, map[string]interface{}{"role": "user", "content": content, "name": ctx.Sid})

	// Add prepare messages witch is query from vector database
	preparePrompts := neo.prepare(ctx, messages)
	if len(preparePrompts) > 0 {
		messages = preparePrompts
	}

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

// NewAI create a new AI
func (neo *DSL) newAI() error {

	if neo.Connector == "" || strings.HasPrefix(neo.Connector, "moapi") {
		model := "gpt-3.5-turbo"
		if strings.HasPrefix(neo.Connector, "moapi:") {
			model = strings.TrimPrefix(neo.Connector, "moapi:")
		}

		ai, err := openai.NewMoapi(model)
		if err != nil {
			return err
		}

		neo.AI = ai
		return nil
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

// Select select the model
func (neo *DSL) Select(model string) error {
	ai, err := openai.NewMoapi(model)
	if err != nil {
		return err
	}
	neo.AI = ai
	return nil
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
