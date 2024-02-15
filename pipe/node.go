package pipe

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openai"
	"github.com/yaoapp/yao/pipe/ui/cli"
)

// Case Execute the user input
func (node *Node) Case(ctx *Context, input Input) (any, error) {

	if node.Switch == nil || len(node.Switch) == 0 {
		return nil, node.Errorf(ctx, "switch case not found")
	}

	input, err := ctx.parseNodeInput(node, input)
	if err != nil {
		return nil, err
	}

	// Find the case
	var child *Pipe = node.Switch["default"]
	data := ctx.data(node)

	for expr, pip := range node.Switch {

		expr, err := data.replaceString(expr)
		if err != nil {
			return nil, err
		}

		v, err := data.Exec(expr)
		if err != nil {
			return nil, err
		}

		if v == true {
			child = pip
		}
	}

	if child == nil {
		return nil, node.Errorf(ctx, "switch case not found")
	}

	// Execute the child pipe
	var res any = nil
	subctx := child.Create().inheritance(ctx)
	if subctx.current != nil {
		res, err = subctx.Exec(input...)
		if err != nil {
			return nil, err
		}
	}

	output, err := ctx.parseNodeOutput(node, res)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// YaoProcess Execute the Yao Process
func (node *Node) YaoProcess(ctx *Context, input Input) (any, error) {

	if node.Process == nil {
		return nil, node.Errorf(ctx, "process not set")
	}

	input, err := ctx.parseNodeInput(node, input)
	if err != nil {
		return nil, err
	}

	data := ctx.data(node)
	args, err := data.replaceArray(node.Process.Args)

	// Execute the process
	process, err := process.Of(node.Process.Name, args...)
	if err != nil {
		return nil, node.Errorf(ctx, err.Error())
	}

	res, err := process.WithGlobal(ctx.global).WithSID(ctx.sid).Exec()
	if err != nil {
		return nil, node.Errorf(ctx, err.Error())
	}

	output, err := ctx.parseNodeOutput(node, res)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// AI Execute the AI input
func (node *Node) AI(ctx *Context, input Input) (any, error) {

	if node.Prompts == nil || len(node.Prompts) == 0 {
		return nil, node.Errorf(ctx, "prompts not found")
	}

	input, err := ctx.parseNodeInput(node, input)
	if err != nil {
		return nil, err
	}

	data := ctx.data(node)
	prompts, err := data.replacePrompts(node.Prompts)
	if err != nil {
		return nil, err
	}
	prompts = node.aiMergeHistory(ctx, prompts)

	res, err := node.chatCompletions(ctx, prompts, node.Options)
	if err != nil {
		return nil, err
	}

	output, err := ctx.parseNodeOutput(node, res)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (node *Node) chatCompletions(ctx *Context, prompts []Prompt, options map[string]interface{}) (any, error) {
	// moapi call
	ai, err := openai.NewMoapi(node.Model)
	if err != nil {
		return nil, err
	}

	response := []string{}
	content := []string{}
	_, ex := ai.ChatCompletions(promptsToMap(prompts), node.Options, func(data []byte) int {

		// Prograss Hook

		if len(data) > 5 && string(data[:5]) == "data:" {
			var res ChatCompletionChunk
			err := jsoniter.Unmarshal(data[5:], &res)
			if err != nil {
				return 0
			}
			if len(res.Choices) > 0 {
				response = append(response, res.Choices[0].Delta.Content)
			}
		} else {
			content = append(content, string(data))
		}

		return 1
	})

	if ex != nil {
		return nil, node.Errorf(ctx, "AI error: %s", ex.Message)
	}

	if (len(response) == 0) && (len(content) > 0) {
		return nil, node.Errorf(ctx, "AI error: %s", strings.Join(content, ""))
	}

	raw := strings.Join(response, "")

	// try to parse the response
	var res any
	err = jsoniter.UnmarshalFromString(raw, &res)
	if err != nil {
		return raw, nil
	}

	return res, nil
}

func (node *Node) aiMergeHistory(ctx *Context, prompts []Prompt) []Prompt {
	if ctx.history == nil {
		ctx.history = map[*Node][]Prompt{}
	}
	if ctx.history[node] == nil {
		ctx.history = map[*Node][]Prompt{}
	}
	new := []Prompt{}
	saved := map[string]bool{}

	// filter the prompts
	for _, prompt := range ctx.history[node] {
		saved[prompt.finger()] = true
		new = append(new, prompt)
	}

	for _, prompt := range prompts {
		if saved[prompt.finger()] {
			continue
		}
		new = append(new, prompt)
	}

	// update the history
	ctx.history[node] = new
	return new
}

// Render Execute the user input
func (node *Node) Render(ctx *Context, input Input) (any, bool, error) {

	switch node.UI {

	case "cli":
		output, err := node.renderCli(ctx, input)
		if err != nil {
			return nil, false, err
		}
		return output, false, nil

	default:
		input, err := ctx.parseNodeInput(node, input)
		if err != nil {
			return nil, true, err
		}

		return ResumeContext{
			ID:    ctx.id,
			Input: input,
			Node:  node,
			Data:  ctx.data(node),
			Type:  node.Type,
			UI:    node.UI,
		}, true, nil

	}
}

func (node *Node) renderCli(ctx *Context, input Input) (any, error) {
	input, err := ctx.parseNodeInput(node, input)
	if err != nil {
		return nil, err
	}

	// Set option
	data := ctx.data(node)
	label, err := data.replaceString(node.Label)
	if err != nil {
		return nil, err
	}

	option := &cli.Option{Label: label}
	if node.AutoFill != nil {

		value := fmt.Sprintf("%v", node.AutoFill.Value)
		value, err = data.replaceString(value)
		if value != "" {
			if err != nil {
				return nil, err
			}

			if node.AutoFill.Action == "exit" {
				value = fmt.Sprintf("%s\nexit()\n", value)
			}
			option.Reader = strings.NewReader(value)
		}
	}

	lines, err := cli.New(option).Render(input)
	if err != nil {
		return nil, node.Errorf(ctx, err.Error())
	}

	output, err := ctx.parseNodeOutput(node, lines)
	if err != nil {
		return nil, node.Errorf(ctx, err.Error())
	}
	return output, nil
}

// Errorf format the error message
func (node *Node) Errorf(ctx *Context, format string, a ...any) error {
	message := fmt.Sprintf(format, a...)
	pid := ctx.Pipe.ID
	return fmt.Errorf("pipe: %s nodes[%d](%s) %s (%s)", pid, node.index, node.Name, message, ctx.id)
}
