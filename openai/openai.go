package openai

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/share"
)

// Tiktoken get number of tokens
func Tiktoken(model string, input string) (int, error) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return 0, err
	}
	token := tkm.Encode(input, nil, nil)
	return len(token), nil
}

// OpenAI struct
type OpenAI struct {
	key          string
	model        string
	host         string
	organization string
	maxToken     int
}

// New create a new OpenAI instance by connector id
func New(id string) (*OpenAI, error) {

	// Moapi integration
	if id == "" || strings.HasPrefix(id, "moapi") {
		model := "gpt-3.5-turbo"
		if strings.HasPrefix(id, "moapi:") {
			model = strings.TrimPrefix(id, "moapi:")
		}
		return NewMoapi(model)
	}

	c, err := connector.Select(id)
	if err != nil {
		return nil, err
	}

	if !c.Is(connector.OPENAI) {
		return nil, fmt.Errorf("The connector %s is not a OpenAI connector", id)
	}

	setting := c.Setting()
	return NewOpenAI(setting)
}

// NewOpenAI create a new OpenAI instance by setting
func NewOpenAI(setting map[string]interface{}) (*OpenAI, error) {

	key := ""
	if v, ok := setting["key"].(string); ok {
		key = v
	}

	model := "gpt-3.5-turbo"
	if v, ok := setting["model"].(string); ok {
		model = v
	}

	host := "https://api.openai.com"
	if v, ok := setting["host"].(string); ok {
		host = v
	}

	organization := ""
	if v, ok := setting["organization"].(string); ok {
		organization = v
	}

	maxToken := 2048
	if v, ok := setting["max_token"].(int); ok {
		maxToken = v
	}

	return &OpenAI{
		key:          key,
		model:        model,
		host:         host,
		organization: organization,
		maxToken:     maxToken,
	}, nil
}

// NewMoapi create a new OpenAI instance by model
// Temporarily: change after the moapi is open source
func NewMoapi(model string) (*OpenAI, error) {

	if model == "" {
		model = "gpt-3.5-turbo"
	}

	url := share.MoapiHosts[0]

	if share.App.Moapi.Mirrors != nil {
		url = share.App.Moapi.Mirrors[0]
	}
	key := share.App.Moapi.Secret
	organization := share.App.Moapi.Organization

	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	if key == "" {
		return nil, fmt.Errorf("The moapi secret is empty")
	}

	return &OpenAI{
		key:          key,
		model:        model,
		host:         url,
		organization: organization,
		maxToken:     16384,
	}, nil
}

// Model get the model
func (openai OpenAI) Model() string {
	return openai.model
}

// Completions Creates a completion for the provided prompt and parameters.
// https://platform.openai.com/docs/api-reference/completions/create
func (openai OpenAI) Completions(prompt interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	if option == nil {
		option = map[string]interface{}{}
	}
	option["prompt"] = prompt

	if cb != nil {
		option["stream"] = true
		return nil, openai.stream(context.Background(), "/v1/completions", option, cb)
	}

	option["stream"] = false
	return openai.post("/v1/completions", option)
}

// CompletionsWith Creates a completion for the provided prompt and parameters.
// https://platform.openai.com/docs/api-reference/completions/create
func (openai OpenAI) CompletionsWith(ctx context.Context, prompt interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	if option == nil {
		option = map[string]interface{}{}
	}
	option["prompt"] = prompt

	if cb != nil {
		option["stream"] = true
		return nil, openai.stream(ctx, "/v1/completions", option, cb)
	}

	option["stream"] = false
	return openai.post("/v1/completions", option)
}

// ChatCompletions Creates a model response for the given chat conversation.
// https://platform.openai.com/docs/api-reference/chat/create
func (openai OpenAI) ChatCompletions(messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	if option == nil {
		option = map[string]interface{}{}
	}
	option["messages"] = messages

	if cb != nil {
		option["stream"] = true
		return nil, openai.stream(context.Background(), "/v1/chat/completions", option, cb)
	}

	option["stream"] = false
	return openai.post("/v1/chat/completions", option)
}

// ChatCompletionsWith Creates a model response for the given chat conversation.
// https://platform.openai.com/docs/api-reference/chat/create
func (openai OpenAI) ChatCompletionsWith(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	if option == nil {
		option = map[string]interface{}{}
	}
	option["messages"] = messages

	if cb != nil {
		option["stream"] = true
		return nil, openai.stream(ctx, "/v1/chat/completions", option, cb)
	}

	option["stream"] = false
	return openai.post("/v1/chat/completions", option)
}

// Edits Creates a new edit for the provided input, instruction, and parameters.
// https://platform.openai.com/docs/api-reference/edits/create
func (openai OpenAI) Edits(instruction string, option map[string]interface{}) (interface{}, *exception.Exception) {
	if option == nil {
		option = map[string]interface{}{}
	}
	option["instruction"] = instruction
	return openai.post("/v1/edits", option)
}

// Embeddings Creates an embedding vector representing the input text.
// https://platform.openai.com/docs/api-reference/embeddings/create
func (openai OpenAI) Embeddings(input interface{}, user string) (interface{}, *exception.Exception) {
	payload := map[string]interface{}{"input": input}
	if user != "" {
		payload["user"] = user
	}
	return openai.post("/v1/embeddings", payload)
}

// AudioTranscriptions Transcribes audio into the input language.
// https://platform.openai.com/docs/api-reference/audio/create
func (openai OpenAI) AudioTranscriptions(dataBase64 string, option map[string]interface{}) (interface{}, *exception.Exception) {
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return nil, exception.New("Base64 error :%s", 400, err.Error())
	}

	if option == nil {
		option = map[string]interface{}{}
	}
	return openai.postFile("/v1/audio/transcriptions", map[string][]byte{"file": data}, option)
}

// ImagesGenerations Creates an image given a prompt.
// https://platform.openai.com/docs/api-reference/images
func (openai OpenAI) ImagesGenerations(prompt string, option map[string]interface{}) (interface{}, *exception.Exception) {
	if option == nil {
		option = map[string]interface{}{}
	}

	if option["response_format"] == nil {
		option["response_format"] = "b64_json"
	}

	option["prompt"] = prompt
	return openai.postWithoutModel("/v1/images/generations", option)
}

// ImagesEdits Creates an edited or extended image given an original image and a prompt.
// https://platform.openai.com/docs/api-reference/images/create-edit
func (openai OpenAI) ImagesEdits(imageBase64 string, prompt string, option map[string]interface{}) (interface{}, *exception.Exception) {

	image, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return nil, exception.New("Base64 error :%s", 400, err.Error())
	}

	files := map[string][]byte{"image": image}

	if option == nil {
		option = map[string]interface{}{}
	}

	if maskBase64, ok := option["mask"].(string); ok {
		mask, err := base64.StdEncoding.DecodeString(maskBase64)
		if err != nil {
			return nil, exception.New("Base64 error :%s", 400, err.Error())
		}
		files["mask"] = mask
	}

	if option["response_format"] == nil {
		option["response_format"] = "b64_json"
	}

	option["prompt"] = prompt
	return openai.postFileWithoutModel("/v1/images/edits", files, option)
}

// ImagesVariations Creates a variation of a given image.
// https://platform.openai.com/docs/api-reference/images/create-variation
func (openai OpenAI) ImagesVariations(imageBase64 string, option map[string]interface{}) (interface{}, *exception.Exception) {

	image, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return nil, exception.New("Base64 error :%s", 400, err.Error())
	}

	files := map[string][]byte{"image": image}
	if option == nil {
		option = map[string]interface{}{}
	}

	if option["response_format"] == nil {
		option["response_format"] = "b64_json"
	}

	return openai.postFileWithoutModel("/v1/images/variations", files, option)
}

// Tiktoken get number of tokens
func (openai OpenAI) Tiktoken(input string) (int, error) {
	tkm, err := tiktoken.EncodingForModel(openai.model)
	if err != nil {
		return 0, err
	}
	token := tkm.Encode(input, nil, nil)
	return len(token), nil
}

// MaxToken get max number of tokens
func (openai OpenAI) MaxToken() int {
	return openai.maxToken
}

// GetContent get the content of chat completions
func (openai OpenAI) GetContent(response interface{}) (string, *exception.Exception) {
	if response == nil {
		return "", exception.New("response is nil", 500)
	}

	if data, ok := response.(map[string]interface{}); ok {
		if choices, ok := data["choices"].([]interface{}); ok {
			if len(choices) == 0 {
				return "", exception.New("choices is null, %v", 500, response)
			}

			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						return content, nil
					}
				}
			}
		}
	}

	return "", exception.New("response format error, %#v", 500, response)
}

// Post post request
func (openai OpenAI) Post(path string, payload map[string]interface{}) (interface{}, *exception.Exception) {
	return openai.post(path, payload)
}

// Stream post request
func (openai OpenAI) Stream(ctx context.Context, path string, payload map[string]interface{}, cb func(data []byte) int) *exception.Exception {
	return openai.stream(ctx, path, payload, cb)
}

// post post request
func (openai OpenAI) post(path string, payload map[string]interface{}) (interface{}, *exception.Exception) {

	url := fmt.Sprintf("%s%s", openai.host, path)
	key := fmt.Sprintf("Bearer %s", openai.key)
	payload["model"] = openai.model

	req := http.New(url).
		WithHeader(map[string][]string{"Authorization": {key}})

	res := req.Post(payload)
	if err := openai.isError(res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

// post post request without model
func (openai OpenAI) postWithoutModel(path string, payload map[string]interface{}) (interface{}, *exception.Exception) {

	url := fmt.Sprintf("%s%s", openai.host, path)
	key := fmt.Sprintf("Bearer %s", openai.key)

	req := http.New(url).
		WithHeader(map[string][]string{"Authorization": {key}})

	res := req.Post(payload)
	if err := openai.isError(res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

// post post request with file
func (openai OpenAI) postFile(path string, files map[string][]byte, option map[string]interface{}) (interface{}, *exception.Exception) {

	url := fmt.Sprintf("%s%s", openai.host, path)
	key := fmt.Sprintf("Bearer %s", openai.key)
	option["model"] = openai.model

	req := http.New(url).
		WithHeader(map[string][]string{
			"Authorization": {key},
			"Content-Type":  {"multipart/form-data"},
		})

	for name, data := range files {
		req.AddFileBytes(name, fmt.Sprintf("%s.mp3", name), data)
	}

	res := req.Send("POST", option)
	if err := openai.isError(res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

// post post request with file without model
func (openai OpenAI) postFileWithoutModel(path string, files map[string][]byte, option map[string]interface{}) (interface{}, *exception.Exception) {

	url := fmt.Sprintf("%s%s", openai.host, path)
	key := fmt.Sprintf("Bearer %s", openai.key)

	req := http.New(url).
		WithHeader(map[string][]string{
			"Authorization": {key},
			"Content-Type":  {"multipart/form-data"},
		})

	for name, data := range files {
		req.AddFileBytes(name, fmt.Sprintf("%s.mp3", name), data)
	}

	res := req.Send("POST", option)
	if err := openai.isError(res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

// stream post request
func (openai OpenAI) stream(ctx context.Context, path string, payload map[string]interface{}, cb func(data []byte) int) *exception.Exception {
	url := fmt.Sprintf("%s%s", openai.host, path)
	key := fmt.Sprintf("Bearer %s", openai.key)
	payload["model"] = openai.model
	req := http.New(url)
	err := req.
		WithHeader(map[string][]string{
			"Content-Type":  {"application/json; charset=utf-8"},
			"Authorization": {key},
		}).
		Stream(ctx, "POST", payload, cb)

	if err != nil {
		return exception.New(err.Error(), 500)
	}
	return nil
}

func (openai OpenAI) isError(res *http.Response) *exception.Exception {

	if res.Status != 200 {
		message := "OpenAI Error"
		if v, ok := res.Data.(string); ok {
			message = v
		}
		if data, ok := res.Data.(map[string]interface{}); ok {
			if err, has := data["error"]; has {
				if err, ok := err.(map[string]interface{}); ok {
					if msg, has := err["message"].(string); has {
						message = msg
					}
					if code, has := err["code"].(string); has {
						message = fmt.Sprintf("OpenAI %s %s", code, message)
					}
				}
			}
		}
		return exception.New(message, res.Status)
	}

	return nil
}
