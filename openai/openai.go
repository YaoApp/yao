package openai

import "github.com/pkoukk/tiktoken-go"

// Tiktoken get number of tokens
func Tiktoken(model string, input string) (int, error) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return 0, err
	}
	token := tkm.Encode(input, nil, nil)
	return len(token), nil
}
