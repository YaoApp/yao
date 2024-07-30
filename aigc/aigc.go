package aigc

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/openai"
)

// Autopilots the loaded autopilots
var Autopilots = []string{}

// AIGCs the loaded AIGCs
var AIGCs = map[string]*DSL{}

// Select select the AIGC
func Select(id string) (*DSL, error) {
	if AIGCs[id] == nil {
		return nil, fmt.Errorf("aigc %s not found", id)
	}
	return AIGCs[id], nil
}

// Call the AIGC
func (ai *DSL) Call(content string, user string, option map[string]interface{}) (interface{}, *exception.Exception) {

	messages := []map[string]interface{}{}
	for _, prompt := range ai.Prompts {
		message := map[string]interface{}{"role": prompt.Role, "content": prompt.Content}
		if prompt.Name != "" {
			message["name"] = prompt.Name
		}
		messages = append(messages, message)
	}

	// add the user message
	message := map[string]interface{}{"role": "user", "content": content}
	if user != "" {
		message["user"] = user
	}
	messages = append(messages, message)

	bytes, err := jsoniter.Marshal(messages)
	if err != nil {
		return nil, exception.New(err.Error(), 400)
	}

	token, err := ai.AI.Tiktoken(string(bytes))
	if err != nil {
		return nil, exception.New(err.Error(), 400)
	}

	if token > ai.AI.MaxToken() {
		return nil, exception.New("token limit exceeded", 400)
	}

	// call the AI
	res, ex := ai.AI.ChatCompletions(messages, option, nil)
	if ex != nil {
		return nil, ex
	}

	resText, ex := ai.AI.GetContent(res)
	if ex != nil {
		return nil, ex
	}

	if ai.Process == "" {
		return resText, nil
	}

	var param interface{} = resText
	if ai.Optional.JSON {
		err = jsoniter.Unmarshal([]byte(resText), &param)
		if err != nil {
			return nil, exception.New("%s parse error: %s", 400, resText, err.Error())
		}
	}

	p, err := process.Of(ai.Process, param)
	if err != nil {
		return nil, exception.New(err.Error(), 400)
	}

	resProcess, err := p.Exec()
	if err != nil {
		return nil, exception.New(err.Error(), 500)
	}

	return resProcess, nil
}

// NewAI create a new AI
func (ai *DSL) newAI() (AI, error) {

	if ai.Connector == "" || strings.HasPrefix(ai.Connector, "moapi") {
		model := "gpt-3.5-turbo"
		if strings.HasPrefix(ai.Connector, "moapi:") {
			model = strings.TrimPrefix(ai.Connector, "moapi:")
		}

		mo, err := openai.NewMoapi(model)
		if err != nil {
			return nil, err
		}
		return mo, nil
	}

	conn, err := connector.Select(ai.Connector)
	if err != nil {
		return nil, err
	}

	if conn.Is(connector.OPENAI) {
		return openai.New(ai.Connector)
	}

	return nil, fmt.Errorf("%s connector %s not support, should be a openai", ai.ID, ai.Connector)
}
