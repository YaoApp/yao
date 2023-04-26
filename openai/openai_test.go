package openai

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/connector"
	"github.com/yaoapp/yao/test"
)

func TestCompletions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "text-davinci-003")
	data, err := openai.Completions("Hello", nil, nil)
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["id"])

	data, err = openai.Completions("Hello", map[string]interface{}{"max_tokens": 2}, nil)
	if err != nil {
		t.Fatal(err.Message)
	}

	usage := data.(map[string]interface{})["usage"].(map[string]interface{})
	assert.Equal(t, 2, int(usage["completion_tokens"].(float64)))

	res := []byte{}
	_, err = openai.Completions("Hello", nil, func(data []byte) int {
		res = append(res, data...)
		if len(data) == 0 {
			res = append(res, []byte("\n")...)
		}

		if string(data) == "data: [DONE]" {
			return 0
		}

		return 1
	})

	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, res)
}

func TestChatCompletions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "gpt-3_5-turbo")
	data, err := openai.ChatCompletions([]map[string]interface{}{{"role": "user", "content": "hello"}}, nil, nil)
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["id"])

	data, err = openai.ChatCompletions([]map[string]interface{}{{"role": "user", "content": "hello"}}, map[string]interface{}{"max_tokens": 2}, nil)
	if err != nil {
		t.Fatal(err.Message)
	}

	usage := data.(map[string]interface{})["usage"].(map[string]interface{})
	assert.Equal(t, 2, int(usage["completion_tokens"].(float64)))

	res := []byte{}
	_, err = openai.ChatCompletions([]map[string]interface{}{{"role": "user", "content": "hello"}}, nil, func(data []byte) int {
		res = append(res, data...)
		if len(data) == 0 {
			res = append(res, []byte("\n")...)
		}

		if string(data) == "data: [DONE]" {
			return 0
		}

		return 1
	})

	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, res)
}

func TestEdits(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "text-davinci-edit-001")
	data, err := openai.Edits("Hello world"+uuid.NewString(), nil)
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["created"])

	data, err = openai.Edits("Fix the spelling mistakes 2nd"+uuid.NewString(), map[string]interface{}{"input": "What day of the wek is it?"})
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["created"])

}

func TestEmbeddings(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "text-embedding-ada-002")
	data, err := openai.Embeddings("The food was delicious and the waiter", "")
	if err != nil {
		t.Fatal(err.Message)
	}

	assert.NotNil(t, data.(map[string]interface{})["data"])

	data, err = openai.Embeddings([]string{"The food was delicious and the waiter", "hello"}, "user-01")
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["data"])
}

func TestAudioTranscriptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "whisper-1")
	data, err := openai.AudioTranscriptions(audio(), nil)
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.Equal(t, "今晚打老虎", data.(map[string]interface{})["text"])
}

// ProcessTiktoken get number of tokens
func TestTiktoken(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "gpt-3_5-turbo")
	res, err := openai.Tiktoken("hello world")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, res)

	res, err = openai.Tiktoken("你好世界！")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 6, res)
}

func prepare(t *testing.T, id string) *OpenAI {
	err := connector.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	openai, err := New(id)
	if err != nil {
		t.Fatal(err)
	}

	return openai
}

func audio() string {
	return `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qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqpMQU1FMy4xMDCqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq//NIxHwAAANIAAAAAKqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq`
}
