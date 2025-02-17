package rules

import (
	gouProcess "github.com/yaoapp/gou/process"
)

func exportProcess() {
	gouProcess.Register("yao.rule.menus", processMenu)
	gouProcess.Register("yao.rule.ruleKeys", processRuleKeys)
}

func processMenu(process *gouProcess.Process) interface{} {
	argsLen := process.NumOfArgs()
	menuOnly := false
	dsls := GetMainKeys()
	menuKeys := []string{"*"}
	if argsLen == 1 {
		dsls = process.ArgsStrings(0)
	} else if argsLen == 2 {
		dsls = process.ArgsStrings(0)
		menuOnly = process.ArgsBool(1)
	} else if argsLen == 3 {
		dsls = process.ArgsStrings(0)
		menuOnly = process.ArgsBool(1)
		menuKeys = process.ArgsStrings(2)
	}
	return GetDSLsMaps(dsls, menuOnly, menuKeys)
}

func processRuleKeys(process *gouProcess.Process) interface{} {
	return GetAllKeys()
}
