package importer

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {

	// 注册处理器
	process.Register("xiang.import.Run", ProcessRun)                       // deprecated → yao.import.Run
	process.Register("xiang.import.Data", ProcessData)                     // deprecated → yao.import.Data
	process.Register("xiang.import.Setting", ProcessSetting)               // deprecated → yao.import.Setting
	process.Register("xiang.import.DataSetting", ProcessDataSetting)       // deprecated → yao.import.DataSetting
	process.Register("xiang.import.Mapping", ProcessMapping)               // deprecated → yao.import.Mapping
	process.Register("xiang.import.MappingSetting", ProcessMappingSetting) // deprecated → yao.import.MappingSetting

	process.Alias("xiang.import.Run", "yao.import.Run")
	process.Alias("xiang.import.Data", "yao.import.Data")
	process.Alias("xiang.import.Setting", "yao.import.Setting")
	process.Alias("xiang.import.DataSetting", "yao.import.DataSetting")
	process.Alias("xiang.import.Mapping", "yao.import.Mapping")
	process.Alias("xiang.import.MappingSetting", "yao.import.MappingSetting")
}

// ProcessRun xiang.import.Run
// 导入数据
func ProcessRun(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	name := process.ArgsString(0)
	imp := Select(name).WithSid(process.Sid)
	filename := process.ArgsString(1)
	src := Open(filename)
	defer src.Close()
	mapping := anyToMapping(process.Args[2])
	return imp.Run(src, mapping)
}

// ProcessSetting xiang.import.Setting
// 导入配置选项
func ProcessSetting(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	imp := Select(name).WithSid(process.Sid)
	return map[string]interface{}{
		"mappingPreview": imp.Option.MappingPreview,
		"dataPreview":    imp.Option.DataPreview,
		"templateLink":   imp.Option.TemplateLink,
		"title":          imp.Title,
	}
}

// ProcessData xiang.import.Data
// 数据预览
func ProcessData(process *process.Process) interface{} {
	process.ValidateArgNums(5)
	name := process.ArgsString(0)
	imp := Select(name).WithSid(process.Sid)

	filename := process.ArgsString(1)
	src := Open(filename)
	defer src.Close()

	page := process.ArgsInt(2)
	size := process.ArgsInt(3)
	mapping := anyToMapping(process.Args[4])

	return imp.DataPreview(src, page, size, mapping)
}

// ProcessDataSetting xiang.import.DataSetting
// 数据预览表格配置
func ProcessDataSetting(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	imp := Select(name).WithSid(process.Sid)
	return imp.DataSetting()
}

// ProcessMapping xiang.import.Mapping
// 字段映射预览
func ProcessMapping(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	imp := Select(name).WithSid(process.Sid)

	filename := process.ArgsString(1)
	src := Open(filename)
	defer src.Close()
	return imp.MappingPreview(src)
}

// ProcessMappingSetting xiang.import.MappingSetting
// 字段映射表格配置
func ProcessMappingSetting(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	imp := Select(name).WithSid(process.Sid)

	filename := process.ArgsString(1)
	src := Open(filename)
	defer src.Close()
	return imp.MappingSetting(src)
}

// 转换为映射表
func anyToMapping(v interface{}) *Mapping {
	var mapping Mapping
	bytes, err := jsoniter.Marshal(v)
	if err != nil {
		exception.New("字段映射表数据格式不正确", 400).Throw()
	}

	err = jsoniter.Unmarshal(bytes, &mapping)
	if err != nil {
		exception.New("字段映射表数据格式不正确", 400).Throw()
	}

	return &mapping
}
