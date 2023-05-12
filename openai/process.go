package openai

import (
	"context"

	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

func init() {
	process.RegisterGroup("openai", map[string]process.Handler{
		"tiktoken":             ProcessTiktoken,
		"embeddings":           ProcessEmbeddings,
		"chat.completions":     ProcessChatCompletions,
		"audio.transcriptions": ProcessAudioTranscriptions,
	})
}

// ProcessTiktoken openai.Tiktoken
func ProcessTiktoken(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	model := process.ArgsString(0)
	input := process.ArgsString(1)
	nums, err := Tiktoken(model, input)
	if err != nil {
		exception.New("Tiktoken error: %s", 400, err).Throw()
	}
	return nums
}

// ProcessEmbeddings openai.Embeddings
func ProcessEmbeddings(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	model := process.ArgsString(0)
	input := process.Args[1]
	user := ""
	if process.NumOfArgs() > 2 {
		user = process.ArgsString(2)
	}

	ai, err := New(model)
	if err != nil {
		exception.New("ChatCompletions error: %s", 400, err).Throw()
	}

	res, ex := ai.Embeddings(input, user)
	if ex != nil {
		ex.Throw()
	}
	return res
}

// ProcessAudioTranscriptions openai.audio.Transcriptions
func ProcessAudioTranscriptions(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	model := process.ArgsString(0)
	dataBase64 := process.ArgsString(1)
	options := map[string]interface{}{}
	if process.NumOfArgs() > 2 {
		if opts, ok := process.Args[2].(map[string]interface{}); ok {
			options = opts
		}
	}

	ai, err := New(model)
	if err != nil {
		exception.New("ChatCompletions error: %s", 400, err).Throw()
	}

	res, ex := ai.AudioTranscriptions(dataBase64, options)
	if ex != nil {
		ex.Throw()
	}
	return res
}

// ProcessChatCompletions openai.chat.Completions
func ProcessChatCompletions(process *process.Process) interface{} {

	process.ValidateArgNums(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	model := process.ArgsString(0)
	messages := []map[string]interface{}{}
	intput := process.ArgsArray(1)
	for idx, v := range intput {
		message, ok := v.(map[string]interface{})
		if !ok {
			exception.New("ChatCompletions input must be array of map, index %d", 400, idx).Throw()
		}
		messages = append(messages, message)
	}

	ai, err := New(model)
	if err != nil {
		exception.New("ChatCompletions error: %s", 400, err).Throw()
	}

	options := map[string]interface{}{}
	if process.NumOfArgs() > 2 {
		if opts, ok := process.Args[2].(map[string]interface{}); ok {
			options = opts
		}
	}

	if process.NumOfArgs() == 3 {
		data, ex := ai.ChatCompletionsWith(ctx, messages, options, nil)
		if ex != nil {
			ex.Throw()
		}
		return data
	}

	if process.NumOfArgs() == 4 {

		switch cb := process.Args[3].(type) {
		case func(data []byte) int:
			res, ex := ai.ChatCompletionsWith(ctx, messages, options, cb)
			if ex != nil {
				ex.Throw()
			}
			return res

		case bridge.FunctionT:
			res, ex := ai.ChatCompletionsWith(ctx, messages, options, func(data []byte) int {

				v, err := cb.Call(string(data))
				if err != nil {
					log.Error("Call callback function error: %s", err.Error())
					return http.HandlerReturnError
				}

				ret, ok := v.(int)
				if !ok {
					log.Error("Callback function must return int")
					return http.HandlerReturnError
				}

				return ret
			})

			if ex != nil {
				ex.Throw()
			}
			return res

		default:
			exception.New("ChatCompletions error: invalid callback arguments", 400).Throw()
			return nil
		}
	}

	res, ex := ai.ChatCompletionsWith(ctx, messages, options, nil)
	if ex != nil {
		ex.Throw()
	}
	return res

}
