package action

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/widgets/hook"
)

// Bind the process name
func (p *Process) Bind(name string) {
	p.ProcessBind = name
}

// NewProcess create a new process
func NewProcess(name string, p *Process) *Process {
	if p == nil {
		p = &Process{}
	}
	p.Name = name
	return p
}

// SetHandler set the handler
func (p *Process) SetHandler(handler Handler) *Process {
	p.Handler = handler
	return p
}

// SetDefault set the default value
func (p *Process) SetDefault(defaults map[string]*Process) *Process {

	if defaultProcess, has := defaults[p.Name]; has {

		p.Name = defaultProcess.Name

		if p.Process == "" {
			p.Process = defaultProcess.Process
		}

		if p.Guard == "" {
			p.Guard = defaultProcess.Guard
		}

		if p.Default == nil {
			p.Default = defaultProcess.Default
		}

		// format defaults
		if len(p.Default) != len(defaultProcess.Default) {
			defauts := defaultProcess.Default
			nums := len(p.Default)
			if nums > len(defaultProcess.Default) {
				nums = len(defaultProcess.Default)
			}
			for i := 0; i < nums; i++ {
				defauts[i] = p.Default[i]
			}
			p.Default = defauts
		}
	}

	return p
}

// WithBefore bind before hook
func (p *Process) WithBefore(before *hook.Before) *Process {
	p.Before = before
	return p
}

// WithAfter bind after hook
func (p *Process) WithAfter(after *hook.After) *Process {
	p.After = after
	return p
}

// Args get the process args
func (p *Process) Args(process *gou.Process) []interface{} {
	process.ValidateArgNums(1)
	args := p.Default
	nums := len(process.Args[1:])
	if nums > len(args) {
		nums = len(args)
	}

	for i := 0; i < nums; i++ {
		args[i] = p.deepMergeDefault(process.Args[i+1], args[i])
	}
	return args
}

// Exec exec the process
func (p *Process) Exec(process *gou.Process) (interface{}, error) {
	if p.Handler == nil {
		return nil, fmt.Errorf("%s handler does not set", p.Name)
	}
	return p.Handler(p, process)
}

// MustExec exec the process
func (p *Process) MustExec(process *gou.Process) interface{} {
	res, err := p.Exec(process)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return res
}

// deepMergeDefault deep merge args
func (p *Process) deepMergeDefault(value interface{}, defaults interface{}) interface{} {

	if value == nil {
		return defaults
	}

	switch defaults.(type) {

	case map[string]interface{}:
		defaultMap := defaults.(map[string]interface{})
		valueMap := any.Of(value).MapStr()
		for key, v := range defaultMap {
			valueMap[key] = p.deepMergeDefault(valueMap[key], v)
		}
		return valueMap

	case []interface{}:
		defaultArr := defaults.([]interface{})
		valueArr := any.Of(value).CArray()

		// pad
		nums := len(defaultArr) - len(valueArr)
		for i := 0; i < nums; i++ {
			valueArr = append(valueArr, nil)
		}

		// set default
		for idx, v := range defaultArr {
			valueArr[idx] = p.deepMergeDefault(valueArr[idx], v)
		}

		return valueArr
	}

	return value
}
