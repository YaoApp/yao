package openai

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
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

func TestCompletionsWith(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "text-davinci-003")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	res := []byte{}
	_, err := openai.CompletionsWith(ctx, "Write an article about internet ", nil, func(data []byte) int {
		res = append(res, data...)
		if len(data) == 0 {
			res = append(res, []byte("\n")...)
		}

		if string(data) == "data: [DONE]" {
			return 0
		}

		return 1
	})

	assert.Contains(t, err.Message, "context canceled")
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

func TestChatCompletionsWith(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "gpt-3_5-turbo")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	res := []byte{}
	_, err := openai.ChatCompletionsWith(ctx, []map[string]interface{}{{"role": "user", "content": "Write an article about internet"}}, nil, func(data []byte) int {
		res = append(res, data...)
		if len(data) == 0 {
			res = append(res, []byte("\n")...)
		}

		if string(data) == "data: [DONE]" {
			return 0
		}

		return 1
	})

	assert.Contains(t, err.Message, "context canceled")
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
	data, err := openai.AudioTranscriptions(audio(t), nil)
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.Equal(t, "今晚打老虎", data.(map[string]interface{})["text"])
}

func TestImagesGenerations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "gpt-3_5-turbo")
	data, err := openai.ImagesGenerations("A cute baby sea otter", nil)
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["created"])

	data, err = openai.ImagesGenerations("A cat", map[string]interface{}{"size": "256x256", "n": 1})
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["created"])
}

func TestImageEdits(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "gpt-3_5-turbo")
	data, err := openai.ImagesEdits(image(t), "change to green", map[string]interface{}{"mask": mask(t)})
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["created"])
}

func TestImageVariations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	openai := prepare(t, "gpt-3_5-turbo")
	data, err := openai.ImagesVariations(image(t), map[string]interface{}{})
	if err != nil {
		t.Fatal(err.Message)
	}
	assert.NotNil(t, data.(map[string]interface{})["created"])
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

func mask(t *testing.T) string {
	fs := fs.MustGet("system")
	data, err := fs.ReadFile("/assets/image_edit_mask.png")
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func image(t *testing.T) string {
	fs := fs.MustGet("system")
	data, err := fs.ReadFile("/assets/image_edit_original.png")
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func audio(t *testing.T) string {
	fs := fs.MustGet("system")
	data, err := fs.ReadFile("/assets/audio_transcriptions.mp3")
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(data)
}
