package neo

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/neo/assistant"
	chatctx "github.com/yaoapp/yao/neo/context"
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
	// res, err := neo.HookCreate(ctx, []map[string]interface{}{}, c)
	// if err != nil {
	// 	return nil, err
	// }

	// Select Assistant
	ast, err := neo.Select(neo.Use.Default)
	if err != nil {
		return nil, err
	}

	// Download file using the assistant
	return ast.Download(ctx.Context, fileID)
}
