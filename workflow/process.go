package workflow

import "github.com/yaoapp/gou"

// WARNING: the Workflow widget will be removed from yao engine

// Process
// 读取工作流 xiang.workflow.Find(name, workflow_id)
// 读取工作流 xiang.workflow.Open(name, uid, data_id)
// 保存工作流 xiang.workflow.Save(name, uid, node_name, data_id, input, ...output)
// 进入下一个节点 xiang.workflow.Next(name, uid, workflow_id, output)
// 跳转到指定节点 xiang.workflow.Goto(name, uid, workflow_id, node_name, output)
// 更新工作流状态 xiang.workflow.Status(name, uid, workflow_id, status_name, output)
// 标记结束流程 xiang.workflow.Done(name, uid, workflow_id, output)
// 标记关闭流程 xiang.workflow.Close(name, uid, workflow_id, output)
// 标记重置流程 xiang.workflow.Reset(name, uid, workflow_id, output)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.workflow.Find", ProcessFind)
	gou.RegisterProcessHandler("xiang.workflow.Setting", ProcessSetting)
	gou.RegisterProcessHandler("xiang.workflow.Open", ProcessOpen)
	gou.RegisterProcessHandler("xiang.workflow.Save", ProcessSave)
	gou.RegisterProcessHandler("xiang.workflow.Next", ProcessNext)
	gou.RegisterProcessHandler("xiang.workflow.Goto", ProcessGoto)
	gou.RegisterProcessHandler("xiang.workflow.Status", ProcessStatus)
	gou.RegisterProcessHandler("xiang.workflow.Done", ProcessDone)
	gou.RegisterProcessHandler("xiang.workflow.Close", ProcessClose)
	gou.RegisterProcessHandler("xiang.workflow.Reset", ProcessReset)
}

// ProcessFind xiang.workflow.Find 读取工作流
//
//	args: [工作流名称*, 工作流ID*]
//
// return: map[string]interface{} 工作流数据记录
func ProcessFind(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	wflow := Select(process.ArgsString(0))
	return wflow.Find(process.ArgsInt(1))
}

// ProcessOpen xiang.workflow.Open 读取工作流
//
//	args: [工作流名称*, 当前用户ID*, 关联数据ID*]
//
// return: map[string]interface{} 工作流数据记录
func ProcessOpen(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	wflow := Select(process.ArgsString(0))
	return wflow.Open(process.ArgsInt(1), process.Args[2])
}

// ProcessSetting xiang.workflow.Setting 读取工作流配置
//
//	args: [工作流名称*, 当前用户ID*, 关联数据ID*]
//
// return: map[string]interface{} 工作流配置
func ProcessSetting(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	wflow := Select(process.ArgsString(0))
	return wflow.Setting(process.ArgsInt(1), process.Args[2])
}

// ProcessSave xiang.workflow.Save 保存工作流节点信息
//
//	args: [工作流名称*, 当前用户ID*, 节点名称*, 关联数据ID*, 输入数据*, 输出数据] (输入数据: {"data":{}, "form":{}}  data 关联数据记录信息, form 工作流body表单信息, 输出数据: {"foo":"bar"} )
//
// return: map[string]interface{} 工作流数据记录
func ProcessSave(process *gou.Process) interface{} {
	process.ValidateArgNums(5)
	wflow := Select(process.ArgsString(0))
	input := InputOf(process.ArgsMap(4))
	output := map[string]interface{}{}
	if process.NumOfArgsIs(6) {
		output = process.ArgsMap(5)
	}
	return wflow.Save(process.ArgsInt(1), process.ArgsString(2), process.Args[3], input, output)
}

// ProcessNext xiang.workflow.Next 进入下一个节点
//
//	args: [工作流名称*, 当前用户ID*, 工作流ID*, 输出数据*] (输出数据: {"foo":"bar"} )
//
// return: map[string]interface{} 工作流数据记录
func ProcessNext(process *gou.Process) interface{} {
	process.ValidateArgNums(4)
	wflow := Select(process.ArgsString(0))
	output := process.ArgsMap(3)
	return wflow.Next(process.ArgsInt(1), process.ArgsInt(2), output)
}

// ProcessGoto xiang.workflow.Goto 跳转到指定节点
//
//	args: [工作流名称*, 当前用户ID*, 工作流ID*, 节点名称*, 输出数据*] (输出数据: {"foo":"bar"} )
//
// return: map[string]interface{} 工作流数据记录
func ProcessGoto(process *gou.Process) interface{} {
	process.ValidateArgNums(5)
	wflow := Select(process.ArgsString(0))
	output := process.ArgsMap(4)
	return wflow.Goto(process.ArgsInt(1), process.ArgsInt(2), process.ArgsString(3), output)
}

// ProcessStatus xiang.workflow.Status 更新工作流状态
//
//	args: [工作流名称*, 当前用户ID*, 工作流ID*, 状态名称*, 输出数据*] (输出数据: {"foo":"bar"} )
//
// return: map[string]interface{} 工作流数据记录
func ProcessStatus(process *gou.Process) interface{} {
	process.ValidateArgNums(5)
	wflow := Select(process.ArgsString(0))
	output := process.ArgsMap(4)
	return wflow.Status(process.ArgsInt(1), process.ArgsInt(2), process.ArgsString(3), output)
}

// ProcessDone xiang.workflow.Done 标记结束流程
//
//	args: [工作流名称*, 当前用户ID*, 工作流ID*, 输出数据*] (输出数据: {"foo":"bar"} )
//
// return: map[string]interface{} 工作流数据记录
func ProcessDone(process *gou.Process) interface{} {
	process.ValidateArgNums(4)
	wflow := Select(process.ArgsString(0))
	output := process.ArgsMap(3)
	return wflow.Done(process.ArgsInt(1), process.ArgsInt(2), output)
}

// ProcessClose xiang.workflow.Close 标记关闭流程
//
//	args: [工作流名称*, 当前用户ID*, 工作流ID*, 输出数据*] (输出数据: {"foo":"bar"} )
//
// return: map[string]interface{} 工作流数据记录
func ProcessClose(process *gou.Process) interface{} {
	process.ValidateArgNums(4)
	wflow := Select(process.ArgsString(0))
	output := process.ArgsMap(3)
	return wflow.Close(process.ArgsInt(1), process.ArgsInt(2), output)
}

// ProcessReset xiang.workflow.Reset 标记重置流程
//
//	args: [工作流名称*, 当前用户ID*, 工作流ID*, 输出数据*] (输出数据: {"foo":"bar"} )
//
// return: map[string]interface{} 工作流数据记录
func ProcessReset(process *gou.Process) interface{} {
	process.ValidateArgNums(4)
	wflow := Select(process.ArgsString(0))
	output := process.ArgsMap(3)
	return wflow.Reset(process.ArgsInt(1), process.ArgsInt(2), output)
}
