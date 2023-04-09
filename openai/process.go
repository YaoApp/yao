package openai

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("yao.openai.tiktoken", ProcessTiktoken)
}

// ProcessTiktoken get number of tokens
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
