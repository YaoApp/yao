package pipe

import (
	"fmt"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/pipe/ui/cli"
)

// ExecProcess Execute the process
func (node Node) ExecProcess(ctx *Context, args []any) error {
	var err error
	name := node.Namespace()
	ctx.in[name] = args
	if node.Input != nil {
		ctx.in[name], err = ctx.replaceInput(node.Input)
		if err != nil {
			return err
		}
	}

	ctx.input[name] = ctx.in[name]
	res := true

	ctx.out[name] = res
	ctx.output[name] = res
	if node.Output != nil {
		ctx.output[name], err = ctx.replace(node.Output)
		if err != nil {
			return err
		}
	}

	next, err := ctx.Next()
	if err != nil {
		if IsEOF(err) {
			return nil
		}
		return err
	}

	// Execute the next node
	_, err = ctx.exec(next, ctx.output[name])
	if err != nil {
		return err
	}
	return nil
}

// ExecRequest Execute the request
func (node Node) ExecRequest(ctx *Context, args []any) error {
	return nil
}

// ExecAI Execute the AI
func (node Node) ExecAI(ctx *Context, args []any) error {
	var err error
	name := node.Namespace()
	ctx.in[name] = args
	if node.Input != nil {
		ctx.in[name], err = ctx.replaceInput(node.Input)
		if err != nil {
			return err
		}
	}
	ctx.input[name] = ctx.in[name]

	res := map[string]any{"args": args, "Chinese": "你好", "Arabic": "مرحبا"}
	ctx.out[name] = res
	ctx.output[name] = res
	if node.Output != nil {
		ctx.output[name], err = ctx.replace(node.Output)
		if err != nil {
			return err
		}
	}

	next, err := ctx.Next()
	if err != nil {
		if IsEOF(err) {
			return nil
		}
		return err
	}

	// Execute the next node
	_, err = ctx.exec(next, ctx.output[name])
	if err != nil {
		return err
	}
	return nil
}

// ExecSwitch Execute the switch
func (node Node) ExecSwitch(ctx *Context, args []any) error {
	var err error
	name := node.Namespace()

	ctx.in[name] = args
	if node.Input != nil {
		ctx.in[name], err = ctx.replaceInput(node.Input)
		if err != nil {
			return err
		}
	}
	ctx.input[name] = ctx.in[name]

	data, err := ctx.data()
	if err != nil {
		return err
	}

	section, _ := node.Case["default"]
	for stmt := range node.Case {
		if stmt == "default" {
			continue
		}

		v, err := data.Exec(stmt)
		if err != nil {
			log.Warn("pipe: %s %s", ctx.Name, err)
			continue
		}

		// If the result is true, then break the loop
		if match, ok := v.(bool); ok && match {
			section = node.Case[stmt]
			break
		}
	}

	// Execute the next node
	if section == nil {
		return fmt.Errorf("pipe: %s %s", ctx.Name, "node case not matched")
	}

	// Execute The Pipe
	subCtx := section.Create().
		With(ctx.context).
		WithGlobal(ctx.global).
		WithSid(ctx.sid)

	// Copy the input and output
	for k, v := range ctx.in {
		subCtx.in[k] = v
	}

	for k, v := range ctx.input {
		subCtx.input[k] = v
	}

	for k, v := range ctx.out {
		subCtx.out[k] = v
	}

	for k, v := range ctx.output {
		subCtx.output[k] = v
	}

	_, err = subCtx.Exec(ctx.in[name])
	if err != nil {
		return err
	}

	// Merge the output
	for k, v := range subCtx.out {
		ctx.out[k] = v
	}

	for k, v := range subCtx.output {
		ctx.output[k] = v
	}

	utils.Dump(name, ctx.output)

	return nil
}

// Render Execute the user input
func (node Node) Render(ctx *Context, args []any) error {

	switch node.UI {

	case "cli":
		return node.renderCli(ctx, args)

	case "web":

	default:
		return fmt.Errorf("pipe: %s %s", ctx.Name, "node ui not supported")
	}

	return nil
}

// Namespace the node namespace
func (node Node) Namespace() string {
	name := node.Name
	if node.namespace != "" {
		name = fmt.Sprintf("%s.%s", node.namespace, name)
	}
	return name
}

func (node Node) renderCli(ctx *Context, args []any) error {

	var err error
	name := node.Namespace()

	ctx.in[name] = args
	if node.Input != nil {
		ctx.in[name], err = ctx.replaceInput(node.Input)
		if err != nil {
			return err
		}
	}
	ctx.input[name] = ctx.in[name]

	// Set option
	label, err := ctx.replaceString(node.Label)
	if err != nil {
		return err
	}

	option := &cli.Option{Label: label}
	if node.AutoFill != nil {

		value := fmt.Sprintf("%v", node.AutoFill.Value)
		value, err = ctx.replaceString(value)
		if value != "" {
			if err != nil {
				fmt.Println("cmd", err)
				return err
			}

			if node.AutoFill.Action == "exit" {
				value = fmt.Sprintf("%s\nexit()\n", value)
			}
			option.Reader = strings.NewReader(value)
		}
	}

	userDataLines, err := cli.New(option).Render(args)
	if err != nil {
		return err
	}

	ctx.out[name] = userDataLines
	ctx.output[name] = userDataLines
	if node.Output != nil {
		ctx.output[name], err = ctx.replace(node.Output)
		if err != nil {
			return err
		}
	}

	// Execute the next node
	next, err := ctx.Next()
	if err != nil {
		if IsEOF(err) {
			return nil
		}
		return err
	}

	// Execute the next node
	_, err = ctx.exec(next, ctx.output[name])
	if err != nil {
		return err
	}

	// Next node
	return nil
}
