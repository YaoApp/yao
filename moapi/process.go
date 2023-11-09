package moapi

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/openai"
)

func init() {
	process.RegisterGroup("moapi", map[string]process.Handler{
		"images.generations": ImagesGenerations,
	})
}

// ImagesGenerations Generate images
func ImagesGenerations(process *process.Process) interface{} {

	process.ValidateArgNums(2)
	model := process.ArgsString(0)
	prompt := process.ArgsString(1)
	option := process.ArgsMap(2, map[string]interface{}{})

	if model == "" {
		exception.New("ImagesGenerations error: model is required", 400).Throw()
	}

	if prompt == "" {
		exception.New("ImagesGenerations error: prompt is required", 400).Throw()
	}

	ai, err := openai.NewMoapi(model)
	if err != nil {
		exception.New("ImagesGenerations error: %s", 400, err).Throw()
	}

	option["prompt"] = prompt
	option["model"] = model
	option["response_format"] = "url"

	res, ex := ai.ImagesGenerations(prompt, option)
	if ex != nil {
		utils.Dump(ex)
		ex.Throw()
	}

	return res
}
