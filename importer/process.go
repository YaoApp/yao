package importer

import "github.com/yaoapp/gou"

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.import.Run", ProcessRun)
	gou.RegisterProcessHandler("xiang.import.Data", ProcessData)
	gou.RegisterProcessHandler("xiang.import.DataSetting", ProcessDataSetting)
	gou.RegisterProcessHandler("xiang.import.Mapping", ProcessMapping)
	gou.RegisterProcessHandler("xiang.import.MappingSetting", ProcessMappingSetting)
	gou.RegisterProcessHandler("xiang.import.Rules", ProcessRules)
}

// ProcessRun xiang.import.Run
// 导入数据
func ProcessRun(process *gou.Process) interface{} {
	return nil
}

// ProcessData xiang.import.Data
// 数据预览
func ProcessData(process *gou.Process) interface{} {
	return nil
}

// ProcessDataSetting xiang.import.DataSetting
// 数据预览表格配置
func ProcessDataSetting(process *gou.Process) interface{} {
	return nil
}

// ProcessMapping xiang.import.Mapping
// 字段映射预览
func ProcessMapping(process *gou.Process) interface{} {
	return nil
}

// ProcessMappingSetting xiang.import.MappingSetting
// 字段映射表格配置
func ProcessMappingSetting(process *gou.Process) interface{} {
	return nil
}

// ProcessRules xiang.import.Rules
// 可用处理器下拉列表
func ProcessRules(process *gou.Process) interface{} {
	return nil
}
