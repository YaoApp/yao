package driver

import (
	"fmt"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/aigc"
	"github.com/yaoapp/yao/neo/command/query"
	"github.com/yaoapp/yao/openai"
)

var commands = sync.Map{}
var requests = sync.Map{}

// Memory the memory driver
type Memory struct {
	model   string
	ai      aigc.AI
	prompts []aigc.Prompt
}

// NewMemory create a new memory driver
func NewMemory(model string, prompts []aigc.Prompt) (*Memory, error) {

	if prompts == nil || len(prompts) == 0 {
		prompts = []aigc.Prompt{
			{
				Role: "system",
				Content: `
					- Answer my question follow this rules:
					- If it can match the "name" or "description" given to you, reply the "ID" of the matched command; 
					- reply the "ID" only, and do not explain your answer, and do not use punctuation.
					- If no matching command is found, reply me <no related command found>. <No relevant command found>, don't answer redundantly.
				`,
			},
		}
	}

	mem := &Memory{model: model, prompts: prompts}
	ai, err := mem.newAI()
	if err != nil {
		return nil, err
	}
	mem.ai = ai
	return mem, nil
}

// Match match the command data
func (driver *Memory) Match(query query.Param, content string) (string, error) {
	prompts := append([]aigc.Prompt{}, driver.prompts...)
	has := false
	commands.Range(func(key, value interface{}) bool {
		cmd, ok := value.(Command)
		if !ok {
			return true
		}
		if query.MatchAny(cmd.Stack, cmd.Path) {
			has = true
			bytes, err := jsoniter.Marshal(map[string]interface{}{
				"id":          cmd.ID,
				"name":        cmd.Name,
				"description": cmd.Description,
				"args":        cmd.Args,
			})
			if err != nil {
				return true
			}
			prompts = append(prompts, aigc.Prompt{
				Role:    "system",
				Content: string(bytes),
			})
		}
		return true
	})

	if !has {
		return "", fmt.Errorf("no related command found")
	}

	messages := []map[string]interface{}{}
	for _, prompt := range prompts {
		messages = append(messages, map[string]interface{}{
			"role":    prompt.Role,
			"content": prompt.Content,
		})
	}

	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": content,
	})

	prompts = append([]aigc.Prompt{}, driver.prompts...)
	res, ex := driver.ai.ChatCompletions(messages, nil, nil)
	if ex != nil {
		return "", fmt.Errorf(ex.Message)
	}

	bytes, err := jsoniter.Marshal(res)
	if err != nil {
		return "", err
	}

	var data struct {
		Choices []struct{ Message struct{ Content string } }
	}
	err = jsoniter.Unmarshal(bytes, &data)
	if err != nil {
		return "", err
	}

	if len(data.Choices) == 0 {
		return "", fmt.Errorf("no related command found")
	}

	return data.Choices[0].Message.Content, nil
}

// Set Set the command data
func (driver *Memory) Set(id string, cmd Command) error {
	commands.Store(id, cmd)
	return nil
}

// Del delete the command data
func (driver *Memory) Del(id string) {
	commands.Delete(id)
}

// Get the command data
func (driver *Memory) Get(id string) (Command, bool) {
	v, ok := commands.Load(id)
	if !ok {
		return Command{}, false
	}
	cmd, ok := v.(Command)
	if !ok {
		return Command{}, false
	}
	return cmd, true
}

// SetRequest set the command request
func (driver *Memory) SetRequest(sid, id, cid string) error {
	requests.Store(sid, Request{
		ID:  id,
		Cid: cid,
		Sid: sid,
	})
	return nil
}

// GetRequest get the command request
func (driver *Memory) GetRequest(sid string) (string, string, bool) {
	v, ok := requests.Load(sid)
	if !ok {
		return "", "", false
	}

	r, ok := v.(Request)
	if !ok {
		return "", "", false
	}

	return r.ID, r.Cid, true
}

// DelRequest delete the command request
func (driver *Memory) DelRequest(sid string) {
	requests.Delete(sid)
}

// NewAI create a new AI
func (driver *Memory) newAI() (aigc.AI, error) {

	if driver.model == "" {
		return nil, fmt.Errorf("%s connector is required", driver.model)
	}

	conn, err := connector.Select(driver.model)
	if err != nil {
		return nil, err
	}

	if conn.Is(connector.OPENAI) {
		return openai.New(driver.model)
	}

	return nil, fmt.Errorf("connector %s not support, should be a openai", driver.model)
}
