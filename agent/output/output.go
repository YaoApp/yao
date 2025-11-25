package output

import (
	"fmt"

	"github.com/yaoapp/yao/agent/output/adapters/cui"
	"github.com/yaoapp/yao/agent/output/adapters/openai"
	"github.com/yaoapp/yao/agent/output/message"
)

// Accept type constants
const (
	AcceptStandard   = "standard"
	AcceptWebCUI     = "cui-web"
	AccepNativeCUI   = "cui-native"
	AcceptDesktopCUI = "cui-desktop"
)

// Output are the options for the output
type Output struct {
	Writer message.Writer
}

// NewOutput creates a new output based on Accept type
func NewOutput(options message.Options) (*Output, error) {
	var writer message.Writer
	var err error

	// Create writer based on Accept type
	switch options.Accept {
	case AcceptStandard:
		// OpenAI-compatible format
		writer, err = openai.NewWriter(options)

	case AcceptWebCUI, AccepNativeCUI, AcceptDesktopCUI:
		// CUI format
		writer, err = cui.NewWriter(options)

	default:
		// Default to Standard (OpenAI)
		writer, err = openai.NewWriter(options)
	}

	if err != nil {
		return nil, err
	}

	return &Output{
		Writer: writer,
	}, nil
}

// Send sends a single message using the appropriate writer for the context
func (o *Output) Send(msg *message.Message) error {
	return o.Writer.Write(msg)
}

// SendGroup sends a message group using the appropriate writer for the context
func (o *Output) SendGroup(group *message.Group) error {
	return o.Writer.WriteGroup(group)
}

// Flush flushes the writer for the given context
func (o *Output) Flush() error {
	return o.Writer.Flush()
}

// Close closes the writer for the given context
func (o *Output) Close() error {
	return o.Writer.Close()
}

// SendMulti sends multiple messages using the appropriate writer for the context
func (o *Output) SendMulti(messages ...*message.Message) error {
	for _, msg := range messages {
		if err := o.Writer.Write(msg); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}
	return nil
}

// // Send sends a single message using the appropriate writer for the context
// func Send(ctx *context.Context, msg *message.Message) error {
// 	writer, err := GetWriter(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	return writer.Write(msg)
// }

// // SendGroup sends a message group using the appropriate writer for the context
// func SendGroup(ctx *context.Context, group *message.Group) error {
// 	writer, err := GetWriter(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	return writer.WriteGroup(group)
// }

// // GetWriter gets or creates a writer for the given context
// // Writers are cached per context to avoid recreating them
// func GetWriter(ctx *context.Context) (message.Writer, error) {
// 	// Try to get cached writer
// 	writerMutex.RLock()
// 	writer, exists := writerCache[ctx]
// 	writerMutex.RUnlock()

// 	if exists {
// 		return writer, nil
// 	}

// 	// Create new writer
// 	writerMutex.Lock()
// 	defer writerMutex.Unlock()

// 	// Double-check after acquiring write lock
// 	if writer, exists := writerCache[ctx]; exists {
// 		return writer, nil
// 	}

// 	// Create writer based on context.Accept
// 	writer, err := createWriter(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Cache the writer
// 	writerCache[ctx] = writer

// 	return writer, nil
// }

// // createWriter creates a writer based on context.Accept
// func createWriter(ctx *context.Context) (message.Writer, error) {
// 	// If global factory is set, use it
// 	if globalFactory != nil {
// 		return globalFactory.NewWriter(ctx, nil)
// 	}

// 	// Default: create based on Accept type
// 	switch ctx.Accept {
// 	case context.AcceptStandard:
// 		// OpenAI-compatible format
// 		return openai.NewWriter(ctx)

// 	case context.AcceptWebCUI, context.AccepNativeCUI, context.AcceptDesktopCUI:
// 		// CUI format
// 		return cui.NewWriter(ctx)

// 	default:
// 		// Default to Standard
// 		return openai.NewWriter(ctx)
// 	}
// }

// // SetWriterFactory sets a custom writer factory
// // This allows applications to provide their own writer implementations
// func SetWriterFactory(factory message.WriterFactory) {
// 	globalFactory = factory
// }

// // ClearWriterCache clears the writer cache
// // Should be called when contexts are cleaned up
// func ClearWriterCache(ctx *context.Context) {
// 	writerMutex.Lock()
// 	defer writerMutex.Unlock()
// 	delete(writerCache, ctx)
// }

// // ClearAllWriterCache clears all cached writers
// func ClearAllWriterCache() {
// 	writerMutex.Lock()
// 	defer writerMutex.Unlock()
// 	writerCache = make(map[*context.Context]message.Writer)
// }

// // Flush flushes the writer for the given context
// func Flush(ctx *context.Context) error {
// 	writer, err := GetWriter(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	return writer.Flush()
// }

// // Close closes the writer for the given context and removes it from cache
// func Close(ctx *context.Context) error {
// 	writer, err := GetWriter(ctx)
// 	if err != nil {
// 		return err
// 	}

// 	err = writer.Close()
// 	ClearWriterCache(ctx)
// 	return err
// }

// // SendMulti is a convenience function to send multiple messages
// func SendMulti(ctx *context.Context, messages ...*message.Message) error {
// 	writer, err := GetWriter(ctx)
// 	if err != nil {
// 		return err
// 	}

// 	for _, msg := range messages {
// 		if err := writer.Write(msg); err != nil {
// 			return fmt.Errorf("failed to send message: %w", err)
// 		}
// 	}

// 	return nil
// }
