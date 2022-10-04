package chart

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/widgets/action"
)

var processActionDefaults = map[string]*action.Process{

	"Setting": {
		Name:    "yao.chart.Setting",
		Process: "yao.chart.Xgen",
		Default: []interface{}{nil},
	},
	"Component": {
		Name:    "yao.chart.Component",
		Default: []interface{}{nil, nil, nil},
	},
	"Data": {
		Name:    "yao.chart.Data",
		Default: []interface{}{nil},
	},
}

// SetDefaultProcess set the default value of action
func (act *ActionDSL) SetDefaultProcess() {

	act.Setting = action.ProcessOf(act.Setting).
		Merge(processActionDefaults["Setting"]).
		SetHandler(processHandler)

	act.Component = action.ProcessOf(act.Component).
		Merge(processActionDefaults["Component"]).
		SetHandler(processHandler)

	act.Data = action.ProcessOf(act.Data).
		WithBefore(act.BeforeData).WithAfter(act.AfterData).
		Merge(processActionDefaults["Data"]).
		SetHandler(processHandler)

}

func processHandler(p *action.Process, process *gou.Process) (interface{}, error) {

	form, err := Get(process)
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
		log.Error("[chart] %s %s process is required", form.ID, p.Name)
		return nil, fmt.Errorf("[chart] %s %s process is required", form.ID, p.Name)
	}

	// Before Hook
	if p.Before != nil {
		log.Trace("[chart] %s %s before: exec(%v)", form.ID, p.Name, args)
		newArgs, err := p.Before.Exec(args, process.Sid, process.Global)
		if err != nil {
			log.Error("[chart] %s %s before: %s", form.ID, p.Name, err.Error())
		} else {
			log.Trace("[chart] %s %s before: args:%v", form.ID, p.Name, args)
			args = newArgs
		}
	}

	// Execute Process
	act, err := gou.ProcessOf(name, args...)
	if err != nil {
		log.Error("[chart] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[chart] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
	}

	res, err := act.WithGlobal(process.Global).WithSID(process.Sid).Exec()
	if err != nil {
		log.Error("[chart] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[chart] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
	}

	// Compute Out
	switch p.Name {

	case "yao.chart.Data":
		switch res.(type) {
		case map[string]interface{}, maps.MapStr:
			data := any.MapOf(res).MapStrAny
			err := form.computeData(process, data)
			if err != nil {
				log.Error("[chart] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
			}
			res = data
		}
		break
	}

	// After hook
	if p.After != nil {
		log.Trace("[chart] %s %s after: exec(%v)", form.ID, p.Name, res)
		newRes, err := p.After.Exec(res, process.Sid, process.Global)
		if err != nil {
			log.Error("[chart] %s %s after: %s", form.ID, p.Name, err.Error())
		} else {
			log.Trace("[chart] %s %s after: %v", form.ID, p.Name, newRes)
			res = newRes
		}
	}

	return res, nil
}
