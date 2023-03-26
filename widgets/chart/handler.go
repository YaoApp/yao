package chart

import (
	"fmt"

	gouProcess "github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/widgets/action"
)

// ********************************
// * Execute the process of form *
// ********************************
// Life-Circle: Before Hook → Run Process → Compute View → After Hook
// Execute Compute View On: Data
func processHandler(p *action.Process, process *gouProcess.Process) (interface{}, error) {

	chart, err := Get(process)
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
		log.Error("[chart] %s %s process is required", chart.ID, p.Name)
		return nil, fmt.Errorf("[chart] %s %s process is required", chart.ID, p.Name)
	}

	// Compute Filter
	err = chart.ComputeFilter(p.Name, process, args, chart.getFilter())
	if err != nil {
		log.Error("[chart] %s %s Compute Filter Error: %s", chart.ID, p.Name, err.Error())
	}

	// Before Hook
	if p.Before != nil {
		log.Trace("[chart] %s %s before: exec(%v)", chart.ID, p.Name, args)
		newArgs, err := p.Before.Exec(args, process.Sid, process.Global)
		if err != nil {
			log.Error("[chart] %s %s before: %s", chart.ID, p.Name, err.Error())
		} else {
			log.Trace("[chart] %s %s before: args:%v", chart.ID, p.Name, args)
			args = newArgs
		}
	}

	// Execute Process
	act, err := gouProcess.Of(name, args...)
	if err != nil {
		log.Error("[chart] %s %s -> %s %s", chart.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[chart] %s %s -> %s %s", chart.ID, p.Name, name, err.Error())
	}

	res, err := act.WithGlobal(process.Global).WithSID(process.Sid).Exec()
	if err != nil {
		log.Error("[chart] %s %s -> %s %s", chart.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[chart] %s %s -> %s %s", chart.ID, p.Name, name, err.Error())
	}

	// Compute View
	err = chart.ComputeView(p.Name, process, res, chart.getField())
	if err != nil {
		log.Error("[chart] %s %s Compute View Error: %s", chart.ID, p.Name, err.Error())
	}

	// After hook
	if p.After != nil {
		log.Trace("[chart] %s %s after: exec(%v)", chart.ID, p.Name, res)
		newRes, err := p.After.Exec(res, process.Sid, process.Global)
		if err != nil {
			log.Error("[chart] %s %s after: %s", chart.ID, p.Name, err.Error())
		} else {
			log.Trace("[chart] %s %s after: %v", chart.ID, p.Name, newRes)
			res = newRes
		}
	}

	return res, nil
}
