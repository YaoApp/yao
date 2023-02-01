package action

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/widgets/hook"
)

// Bind the process name
func (p *Process) Bind(processName string) {
	p.ProcessBind = processName
}

// SetName set the process name
func (p *Process) SetName(name string) {
	p.Name = name
}

// SetHandler set the handler
func (p *Process) SetHandler(handler Handler) *Process {
	p.Handler = handler
	return p
}

// Merge a process
func (p *Process) Merge(newProcess *Process) *Process {

	if newProcess == nil {
		return p
	}

	if newProcess.Name != "" {
		p.Name = newProcess.Name
	}

	if p.Process == "" {
		p.Process = newProcess.Process
	}

	if p.ProcessBind == "" {
		p.ProcessBind = newProcess.ProcessBind
	}

	if p.Guard == "" {
		p.Guard = newProcess.Guard
	}

	p.DefaultMerge(newProcess.Default)
	return p
}

// DefaultMerge merge the default value.
// option[0] the default is false, if true overwrite by the default value;
// option[1] the default is true,  if true deep merge map and slice;
func (p *Process) DefaultMerge(defaults []interface{}, option ...bool) {

	overwrite := false
	if len(option) > 0 && option[0] {
		overwrite = true
	}

	deep := true
	if len(option) > 1 && !option[1] {
		deep = false
	}

	if defaults == nil {
		return
	}

	if p.Default == nil {
		p.Default = []interface{}{}
	}

	length := len(p.Default)
	for idx, value := range defaults {
		if idx >= length {
			p.Default = append(p.Default, value)
			continue
		}
		if value != nil {
			p.Default[idx] = p.mergeDefaultValue("", p.Default[idx], value, overwrite, deep)
		}
	}
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
func (p *Process) Args(process *process.Process) []interface{} {
	process.ValidateArgNums(1)
	args := []interface{}{}
	args = append(args, p.Default...)
	nums := len(process.Args[1:])
	if nums > len(args) {
		nums = len(args)
	}

	for i := 0; i < nums; i++ {
		input := process.Args[i+1]
		defaultValue := args[i]
		args[i] = p.mergeDefaultValue(process.Sid, input, defaultValue, false, true)
		// fmt.Printf("-Args--\n%#v\n===\n%#v\n-END Args--\n\n", input, args[i])
	}
	return args
}

// Exec exec the process
func (p *Process) Exec(process *process.Process) (interface{}, error) {
	if p.Handler == nil {
		return nil, fmt.Errorf("%s handler does not set", p.Name)
	}
	return p.Handler(p, process)
}

// MustExec exec the process
func (p *Process) MustExec(process *process.Process) interface{} {
	res, err := p.Exec(process)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return res
}

// deepMergeDefault deep merge args
func (p *Process) mergeDefaultValue(sid string, value interface{}, defaultValue interface{}, overwrite bool, deep bool) interface{} {

	switch defaultValue.(type) {

	case map[string]interface{}:
		return p.mergeDefaultMap(sid, value, defaultValue.(map[string]interface{}), overwrite, deep)

	case []interface{}:
		return p.mergeDefaultSlice(sid, value, defaultValue.([]interface{}), overwrite, deep)

	case string:
		return p.mergeDefaultString(sid, value, defaultValue.(string), overwrite)
	}

	if value == nil || overwrite {
		return defaultValue
	}

	if vstr, ok := value.(string); ok && vstr == "" {
		return defaultValue
	} else if vint, ok := value.(int); ok && vint == 0 {
		return defaultValue
	}

	return value
}

func (p *Process) mergeDefaultMap(sid string, value interface{}, defaultValues map[string]interface{}, overwrite bool, deep bool) interface{} {
	if value == nil {
		value = map[string]interface{}{}
		// return defaultValues
	}

	vmap := map[string]interface{}(any.Of(value).Map().MapStrAny)
	// fmt.Printf("-vmap--\n%#v\n-end vmap--\n\n", vmap)

	for k, v := range defaultValues {
		// fmt.Printf("-vmap k:v--\n%#v:%#v\n-end vmap  k:v--\n\n", k, v)

		if _, has := vmap[k]; !has || overwrite || deep {
			if deep {
				vmap[k] = p.mergeDefaultValue(sid, vmap[k], v, overwrite, deep)
				continue
			}
			vmap[k] = v
		}
	}

	// delete keys
	if overwrite && !deep {
		for k := range vmap {
			if _, has := defaultValues[k]; !has {
				delete(vmap, k)
			}
		}
	}

	return vmap
}

func (p *Process) mergeDefaultSlice(sid string, value interface{}, defaultValues []interface{}, overwrite bool, deep bool) interface{} {
	if value == nil {
		value = []interface{}{}
		// return defaultValues
	}

	varr := any.Of(value).CArray()
	// fmt.Printf("-varr--\n%#v\n-end varr--\n\n", defaultValues)

	length := len(varr)
	for i, v := range defaultValues {
		if i >= length {
			if deep {
				varr = append(varr, p.mergeDefaultValue(sid, nil, v, overwrite, deep))
				// fmt.Printf("-varr-deep --\n%#v\n-end varr--\n\n", varr)
				continue
			}

			varr = append(varr, v)
			continue
		}

		if overwrite {
			if deep {
				varr[i] = p.mergeDefaultValue(sid, varr[i], v, overwrite, deep)
			} else {
				varr[i] = v
			}
		}
	}

	// delete keys
	if overwrite && !deep && length > len(defaultValues) {
		varr = varr[:len(defaultValues)]
	}

	return varr
}

func (p *Process) mergeDefaultString(sid string, value interface{}, defaultValue string, overwrite bool) interface{} {

	if value == nil || overwrite {
		value = defaultValue
	}

	if valueStr, ok := value.(string); ok && sid != "" {

		if valueStr == "" {
			return defaultValue
		}

		// Session $.user.id $.user_id
		v := strings.TrimSpace(valueStr)
		if strings.HasPrefix(v, "$.") {
			name := strings.TrimLeft(v, "$.")
			namer := strings.Split(name, ".")

			val, err := session.Global().ID(sid).Get(namer[0])

			if err != nil {
				exception.New("Get %s %s", 500, v, err.Error()).Throw()
				return nil
			}

			// $.user_id
			if len(namer) == 1 {
				log.Trace("[Session] %s %v", v, val)
				return val
			}

			// $.user.id
			mapping := any.Of(val).MapStr().Dot()
			val = mapping.Get(strings.Join(namer[1:], "."))
			log.Trace("[Session] %s %v", v, val)
			return val
		}

	}

	return value
}
