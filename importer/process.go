package importer

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

func init() {

	// 注册处理器
	gou.RegisterProcessHandler("xiang.import.Run", ProcessRun)                       // deprecated → yao.import.Run
	gou.RegisterProcessHandler("xiang.import.Data", ProcessData)                     // deprecated → yao.import.Data
	gou.RegisterProcessHandler("xiang.import.Setting", ProcessSetting)               // deprecated → yao.import.Setting
	gou.RegisterProcessHandler("xiang.import.DataSetting", ProcessDataSetting)       // deprecated → yao.import.DataSetting
	gou.RegisterProcessHandler("xiang.import.Mapping", ProcessMapping)               // deprecated → yao.import.Mapping
	gou.RegisterProcessHandler("xiang.import.MappingSetting", ProcessMappingSetting) // deprecated → yao.import.MappingSetting

	gou.AliasProcess("xiang.import.Run", "yao.import.Run")
	gou.AliasProcess("xiang.import.Data", "yao.import.Data")
	gou.AliasProcess("xiang.import.Setting", "yao.import.Setting")
	gou.AliasProcess("xiang.import.DataSetting", "yao.import.DataSetting")
	gou.AliasProcess("xiang.import.Mapping", "yao.import.Mapping")
	gou.AliasProcess("xiang.import.MappingSetting", "yao.import.MappingSetting")
}

// ProcessRun xiang.import.Run
// 导入数据
func ProcessRun(process *gou.Process) interface{} {
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
func ProcessSetting(process *gou.Process) interface{} {
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
func ProcessData(process *gou.Process) interface{} {
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
func ProcessDataSetting(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	imp := Select(name).WithSid(process.Sid)
	return imp.DataSetting()
}

// ProcessMapping xiang.import.Mapping
// 字段映射预览
func ProcessMapping(process *gou.Process) interface{} {
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
func ProcessMappingSetting(process *gou.Process) interface{} {
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
