package neo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/neo/conversation"
	"github.com/yaoapp/yao/openai"
)

// API is a method on the Neo type
func (neo *Neo) API(router *gin.Engine, path string, allows ...string) error {

	prompts := []map[string]interface{}{}
	for _, prompt := range neo.Prompts {
		message := map[string]interface{}{"role": prompt.Role, "content": prompt.Content}
		if prompt.Name != "" {
			message["name"] = prompt.Name
		}
		prompts = append(prompts, message)
	}

	// set the guard
	err := neo.setGuard(router)
	if err != nil {
		return err
	}

	// Cross-Domain
	neo.crossDomain(router, path, allows...)

	// api router
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

		messages := append([]map[string]interface{}{}, prompts...)
		history, err := neo.Conversation.GetHistory(sid)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
		}

		messages = append(messages, history...)
		messages = append(messages, map[string]interface{}{"role": "user", "content": content, "name": sid})

		// reply the content
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = neo.Answer(ctx, c, messages)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
		}

	})

	return nil
}

// Answer the message
func (neo *Neo) Answer(ctx context.Context, c *gin.Context, messages []map[string]interface{}) error {

	chanStream := make(chan []byte, 1)
	chanError := make(chan error, 1)

	go func() {
		defer func() {
			close(chanStream)
			close(chanError)
		}()

		_, ex := neo.AI.ChatCompletionsWith(ctx, messages, neo.Option, func(data []byte) int {
			chanStream <- data
			return 1
		})

		if ex != nil {
			chanError <- fmt.Errorf("AI chat error: %s", ex.Message)
		}
	}()

	c.Header("Content-Type", "text/event-stream;charset=utf-8")
	ok := c.Stream(func(w io.Writer) bool {
		select {
		case err := <-chanError:
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
			}
			return false

		case msg := <-chanStream:
			msg = append(msg, []byte("\n")...)
			w.Write(msg)
			return true
		}
	})

	if !ok {
		c.Status(500)
		return nil
	}

	c.Status(200)
	return nil
}

func (neo *Neo) crossDomain(router *gin.Engine, path string, allows ...string) {

	if len(allows) == 0 {
		return
	}

	allowsMap := map[string]bool{}
	for _, allow := range allows {
		allowsMap[allow] = true
	}

	router.Use(func(c *gin.Context) {
		referer := c.Request.Referer()
		if referer != "" {

			if !api.IsAllowed(c, allowsMap) {
				c.AbortWithStatus(403)
				return
			}

			url, _ := url.Parse(referer)
			referer = fmt.Sprintf("%s://%s", url.Scheme, url.Host)
			c.Writer.Header().Set("Access-Control-Allow-Origin", referer)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
			c.AbortWithStatus(204)
		}
	})

	router.OPTIONS(path, func(c *gin.Context) { c.Status(200) })
}

func (neo *Neo) setGuard(router *gin.Engine) error {

	if neo.Guard == "" {
		router.Use(func(c *gin.Context) {
			token := c.Query("token")
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
func (neo *Neo) newAI() error {

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
func (neo *Neo) newConversation() error {

	if neo.ConversationSetting.Connector == "default" || neo.ConversationSetting.Connector == "" {
		neo.Conversation = conversation.NewXun()
		return nil
	}

	conn, err := connector.Select(neo.ConversationSetting.Connector)
	if err != nil {
		return err
	}

	if conn.Is(connector.DATABASE) {
		neo.Conversation = conversation.NewXun()
		return nil

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
