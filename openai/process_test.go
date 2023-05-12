package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessTiktoken(t *testing.T) {
	// Hash
	args := []interface{}{"gpt-3.5-turbo", "hello world"}
	res := process.New("openai.Tiktoken", args...).Run()
	assert.Equal(t, 2, res)

	args = []interface{}{"gpt-3.5-turbo", "你好世界！"}
	res = process.New("openai.Tiktoken", args...).Run()
	assert.Equal(t, 6, res)
}

func TestProcessEmbeddings(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	args := []interface{}{"text-embedding-ada-002", "hello world"}
	data := process.New("openai.Embeddings", args...).Run()
	assert.NotNil(t, data.(map[string]interface{})["data"])

	args = []interface{}{"text-embedding-ada-002", []string{"The food was delicious and the waiter", "hello"}, "user-01"}
	data = process.New("openai.Embeddings", args...).Run()
	assert.NotNil(t, data.(map[string]interface{})["data"])
}

func TestProcessAudioTranscriptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	args := []interface{}{"whisper-1", audio(t)}
	data := process.New("openai.audio.Transcriptions", args...).Run()
	assert.Equal(t, "今晚打老虎", data.(map[string]interface{})["text"])
}

func TestProcessChatCompletions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	args := []interface{}{"gpt-3_5-turbo", []map[string]interface{}{{"role": "user", "content": "hello"}}}
	res := process.New("openai.chat.Completions", args...).Run()
	data, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("ChatCompletions return type error")
	}
	assert.NotEmpty(t, data["id"])

	// With options
	args = []interface{}{
		"gpt-3_5-turbo",
		[]map[string]interface{}{{"role": "user", "content": "hello"}},
		map[string]interface{}{"max_tokens": 2},
	}
	res = process.New("openai.chat.Completions", args...).Run()
	data, ok = res.(map[string]interface{})
	if !ok {
		t.Fatalf("ChatCompletions return type error")
	}

	usage, ok := data["usage"].(map[string]interface{})
	if !ok {
		t.Fatalf("ChatCompletions return type error")
	}
	assert.Equal(t, 2, int(usage["completion_tokens"].(float64)))

	// With callback
	content := []byte{}
	args = []interface{}{
		"gpt-3_5-turbo",
		[]map[string]interface{}{{"role": "user", "content": "hello"}},
		nil,
		func(data []byte) int {

			content = append(content, data...)
			if len(data) == 0 {
				res = append(content, []byte("\n")...)
			}

			if string(data) == "data: [DONE]" {
				return 0
			}

			return 1
		},
	}
	res = process.New("openai.chat.Completions", args...).Run()
	assert.Contains(t, string(content), "[DONE]")

	// With JS Callback
	res, err := process.New("scripts.openai.TestProcessChatCompletions").Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, res, "[DONE]")
}
