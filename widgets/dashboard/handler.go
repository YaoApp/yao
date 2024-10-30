package dashboard

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

	dashboard, err := Get(process)
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
		log.Error("[dashboard] %s %s process is required", dashboard.ID, p.Name)
		return nil, fmt.Errorf("[dashboard] %s %s process is required", dashboard.ID, p.Name)
	}

	// Compute Filter
	err = dashboard.ComputeFilter(p.Name, process, args, dashboard.getFilter())
	if err != nil {
		log.Error("[dashboard] %s %s Compute Filter Error: %s", dashboard.ID, p.Name, err.Error())
	}

	// Before Hook
	if p.Before != nil {
		log.Trace("[dashboard] %s %s before: exec(%v)", dashboard.ID, p.Name, args)
		newArgs, err := p.Before.Exec(args, process.Sid, process.Global)
		if err != nil {
			log.Error("[dashboard] %s %s before: %s", dashboard.ID, p.Name, err.Error())
		} else {
			log.Trace("[dashboard] %s %s before: args:%v", dashboard.ID, p.Name, args)
			args = newArgs
		}
	}

	// Execute Process
	act, err := gouProcess.Of(name, args...)
	if err != nil {
		log.Error("[dashboard] %s %s -> %s %s", dashboard.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[dashboard] %s %s -> %s %s", dashboard.ID, p.Name, name, err.Error())
	}

	err = act.WithGlobal(process.Global).WithSID(process.Sid).Execute()
	if err != nil {
		log.Error("[dashboard] %s %s -> %s %s", dashboard.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[dashboard] %s %s -> %s %s", dashboard.ID, p.Name, name, err.Error())
	}
	defer act.Release()
	res := act.Value()

	// Compute View
	err = dashboard.ComputeView(p.Name, process, res, dashboard.getField())
	if err != nil {
		log.Error("[dashboard] %s %s Compute View Error: %s", dashboard.ID, p.Name, err.Error())
	}

	// After hook
	if p.After != nil {
		log.Trace("[dashboard] %s %s after: exec(%v)", dashboard.ID, p.Name, res)
		newRes, err := p.After.Exec(res, process.Sid, process.Global)
		if err != nil {
			log.Error("[dashboard] %s %s after: %s", dashboard.ID, p.Name, err.Error())
		} else {
			log.Trace("[dashboard] %s %s after: %v", dashboard.ID, p.Name, newRes)
			res = newRes
		}
	}

	return res, nil
}
