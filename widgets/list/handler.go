package list

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/model"
	gouProcess "github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/widgets/action"
)

// ********************************
// * Execute the process of list *
// ********************************
// Life-Circle: Before Hook → Compute Edit → Run Process → Compute View → After Hook
// Execute Compute Edit On:    Save
// Execute Compute View On:    Get
func processHandler(p *action.Process, process *gouProcess.Process) (interface{}, error) {

	list, err := Get(process)
	if err != nil {
		return nil, err
	}
	args := p.Args(process)

	// Process
	name := p.Process
	if name == "" {
		name = p.ProcessBind
	}

	if name == "" {
		log.Error("[list] %s %s process is required", list.ID, p.Name)
		return nil, fmt.Errorf("[list] %s %s process is required", list.ID, p.Name)
	}

	// Before Hook
	if p.Before != nil {
		log.Trace("[list] %s %s before: exec(%v)", list.ID, p.Name, args)
		newArgs, err := p.Before.Exec(args, process.Sid, process.Global)
		if err != nil {
			log.Error("[list] %s %s before: %s", list.ID, p.Name, err.Error())
		} else {
			log.Trace("[list] %s %s before: args:%v", list.ID, p.Name, args)
			args = newArgs
		}
	}

	// Compute Edit
	err = list.ComputeEdit(p.Name, process, args, list.getField())
	if err != nil {
		log.Error("[list] %s %s Compute Edit Error: %s", list.ID, p.Name, err.Error())
	}

	// Execute Process
	act, err := gouProcess.Of(name, args...)
	if err != nil {
		log.Error("[list] %s %s -> %s %s", list.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[list] %s %s -> %s %s", list.ID, p.Name, name, err.Error())
	}

	err = act.WithGlobal(process.Global).WithSID(process.Sid).Execute()
	if err != nil {
		log.Error("[list] %s %s -> %s %s", list.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[list] %s %s -> %s %s", list.ID, p.Name, name, err.Error())
	}
	defer act.Release()
	res := act.Value()

	// Compute View
	err = list.ComputeView(p.Name, process, res, list.getField())
	if err != nil {
		log.Error("[list] %s %s Compute View Error: %s", list.ID, p.Name, err.Error())
	}

	// After hook
	if p.After != nil {
		log.Trace("[list] %s %s after: exec(%v)", list.ID, p.Name, res)
		newRes, err := p.After.Exec(res, process.Sid, process.Global)
		if err != nil {
			log.Error("[list] %s %s after: %s", list.ID, p.Name, err.Error())
		} else {
			log.Trace("[list] %s %s after: %v", list.ID, p.Name, newRes)
			res = newRes
		}
	}

	// Tranlate the result
	newRes, err := list.translate(p.Name, process, res)
	if err != nil {
		return nil, fmt.Errorf("[list] %s %s Translate Error: %s", list.ID, p.Name, err.Error())
	}

	return newRes, nil
}

// translateSetting
func (dsl *DSL) translate(name string, process *gouProcess.Process, data interface{}) (interface{}, error) {

	if strings.ToLower(name) != "yao.list.setting" {
		return data, nil
	}

	widgets := []string{}
	if dsl.Action.Bind != nil {
		if dsl.Action.Bind.Model != "" {
			m := model.Select(dsl.Action.Bind.Model)
			widgets = append(widgets, fmt.Sprintf("model.%s", m.ID))
		}

		if dsl.Action.Bind.Table != "" {
			widgets = append(widgets, fmt.Sprintf("table.%s", dsl.Action.Bind.Table))
		}
	}

	widgets = append(widgets, fmt.Sprintf("list.%s", dsl.ID))
	res, err := i18n.Trans(session.Lang(process, config.Conf.Lang), widgets, data)
	if err != nil {
		return nil, err
	}

	return res, nil
}
