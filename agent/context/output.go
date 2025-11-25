package context

import (
	"github.com/yaoapp/yao/agent/output"
	"github.com/yaoapp/yao/agent/output/message"
)

// Send sends a message via the output module
func (ctx *Context) Send(msg *message.Message) error {
	output, err := ctx.getOutput()
	if err != nil {
		return err
	}
	return output.Send(msg)
}

// SendGroup sends a group of messages via the output module
func (ctx *Context) SendGroup(group *message.Group) error {
	output, err := ctx.getOutput()
	if err != nil {
		return err
	}
	return output.SendGroup(group)
}

// Flush flushes the output writer
func (ctx *Context) Flush() error {
	output, err := ctx.getOutput()
	if err != nil {
		return err
	}
	return output.Flush()
}

// CloseOutput closes the output writer
func (ctx *Context) CloseOutput() error {
	output, err := ctx.getOutput()
	if err != nil {
		return err
	}
	return output.Close()
}

// getOutput gets the output writer for the context
func (ctx *Context) getOutput() (*output.Output, error) {
	if ctx.output != nil {
		return ctx.output, nil
	}

	trace, _ := ctx.Trace()
	var options message.Options = message.Options{
		BaseURL: "/",
		Writer:  ctx.Writer,
		Trace:   trace,
		Locale:  ctx.Locale,
		Accept:  string(ctx.Accept),
	}

	// Convert ModelCapabilities to message.ModelCapabilities
	if ctx.Capabilities != nil {
		options.Capabilities = &message.ModelCapabilities{
			Vision:                ctx.Capabilities.Vision,
			ToolCalls:             ctx.Capabilities.ToolCalls,
			Audio:                 ctx.Capabilities.Audio,
			Reasoning:             ctx.Capabilities.Reasoning,
			Streaming:             ctx.Capabilities.Streaming,
			JSON:                  ctx.Capabilities.JSON,
			Multimodal:            ctx.Capabilities.Multimodal,
			TemperatureAdjustable: ctx.Capabilities.TemperatureAdjustable,
		}
	}

	var err error
	ctx.output, err = output.NewOutput(options)
	if err != nil {
		return nil, err
	}
	return ctx.output, nil
}
