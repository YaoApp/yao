package table

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/widgets/action"
)

// ********************************
// * Execute the process of table *
// ********************************
// Life-Circle: Compute Filter → Before Hook → Compute Edit → Run Process → Compute View → After Hook
// Execute Compute Filter On:  Search, Get, Find
// Execute Compute Edit On:    Save, Create, Update, UpdateWhere, UpdateIn, Insert(not support yet)
// Execute Compute View On:    Search, Get, Find
func processHandler(p *action.Process, process *gou.Process) (interface{}, error) {

	tab, err := Get(process)
	if err != nil {
		return nil, fmt.Errorf("[table] %s %s %s", tab.ID, p.Name, err.Error())
	}
	args := p.Args(process)

	// Process
	name := p.Process
	if name == "" {
		name = p.ProcessBind
	}

	if name == "" {
		log.Error("[table] %s %s process is required", tab.ID, p.Name)
		return nil, fmt.Errorf("[table] %s %s process is required", tab.ID, p.Name)
	}

	// Compute Filter
	err = tab.computeFilter(p.Name, process, args)
	if err != nil {
		log.Error("[table] %s %s Compute Filter Error: %s", tab.ID, p.Name, err.Error())
	}

	// Before Hook
	if p.Before != nil {
		log.Trace("[table] %s %s before: exec(%v)", tab.ID, p.Name, args)
		newArgs, err := p.Before.Exec(args, process.Sid, process.Global)
		if err != nil {
			log.Error("[table] %s %s before: %s", tab.ID, p.Name, err.Error())
		} else {
			log.Trace("[table] %s %s before: args:%v", tab.ID, p.Name, args)
			args = newArgs
		}
	}

	// Compute Edit
	err = tab.computeEdit(p.Name, process, args)
	if err != nil {
		log.Error("[table] %s %s Compute Edit Error: %s", tab.ID, p.Name, err.Error())
	}

	// Execute Process
	act, err := gou.ProcessOf(name, args...)
	if err != nil {
		log.Error("[table] %s %s -> %s %s %v", tab.ID, p.Name, name, err.Error(), args)
		return nil, fmt.Errorf("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
	}

	res, err := act.WithGlobal(process.Global).WithSID(process.Sid).Exec()
	if err != nil {
		log.Error("[table] %s %s -> %s %s %v", tab.ID, p.Name, name, err.Error(), args)
		return nil, fmt.Errorf("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
	}

	// Compute View
	err = tab.computeView(p.Name, process, res)
	if err != nil {
		log.Error("[table] %s %s Compute View Error: %s", tab.ID, p.Name, err.Error())
	}

	// After hook
	if p.After != nil {
		log.Trace("[table] %s %s after: exec(%v)", tab.ID, p.Name, res)
		newRes, err := p.After.Exec(res, process.Sid, process.Global)
		if err != nil {
			log.Error("[table] %s %s after: %s", tab.ID, p.Name, err.Error())
		} else {
			log.Trace("[table] %s %s after: %v", tab.ID, p.Name, newRes)
			res = newRes
		}
	}

	// Tranlate the result
	newRes, err := tab.translate(p.Name, process, res)
	if err != nil {
		return nil, fmt.Errorf("[table] %s %s Translate Error: %s", tab.ID, p.Name, err.Error())
	}

	return newRes, nil
}

// translateSetting
func (dsl *DSL) translate(name string, process *gou.Process, data interface{}) (interface{}, error) {

	if name != "yao.table.Setting" {
		return data, nil
	}

	widgets := []string{}
	if dsl.Action.Bind.Model != "" {
		m := gou.Select(dsl.Action.Bind.Model)
		widgets = append(widgets, fmt.Sprintf("model.%s", m.ID))
	}

	if dsl.Action.Bind.Table != "" {
		widgets = append(widgets, fmt.Sprintf("table.%s", dsl.Action.Bind.Table))
	}

	widgets = append(widgets, fmt.Sprintf("table.%s", dsl.ID))
	res, err := i18n.Trans(process.Lang(config.Conf.Lang), widgets, data)
	if err != nil {
		return nil, err
	}

	return res, nil
}
