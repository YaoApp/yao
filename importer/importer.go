package importer

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/importer/from"
	"github.com/yaoapp/xiang/importer/xlsx"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/xfs"
	"github.com/yaoapp/xiang/xlog"
)

// Importers 导入器
var Importers = map[string]*Importer{}

// Load 加载导入器
func Load(cfg config.Config) {
	LoadFrom(filepath.Join(cfg.Root, "imports"), "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) {
	if share.DirNotExists(dir) {
		return
	}
	share.Walk(dir, ".json", func(root, filename string) {
		var importer Importer
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		err := jsoniter.Unmarshal(content, &importer)
		if err != nil {
			exception.New("%s 导入配置错误. %s", 400, name, err.Error()).Ctx(filename).Throw()
		}
		Importers[name] = &importer
	})
}

// Select 选择已加载导入器
func Select(name string) *Importer {
	im, has := Importers[name]
	if !has {
		exception.New("导入配置: %s 尚未加载", 400, name).Throw()
	}
	return im
}

// Open 打开导入内容源
func Open(name string) from.Source {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	switch ext {
	case "xlsx":
		fullpath := name
		if !strings.HasPrefix(fullpath, "/") {
			fullpath = filepath.Join(xfs.Stor.Root, name)
		}
		return xlsx.Open(fullpath)
	}
	exception.New("暂不支持: %s 文件导入", 400, ext).Throw()
	return nil
}

// AutoMapping 根据文件信息获取字段映射表
func (imp *Importer) AutoMapping(src from.Source) *Mapping {
	sourceColumns := getSourceColumns(src)
	sourceInspect := src.Inspect()
	mapping := &Mapping{
		Columns:          []*Binding{},
		AutoMatching:     true,
		TemplateMatching: false,
		Sheet:            sourceInspect.SheetName,
		ColStart:         sourceInspect.ColStart,
		RowStart:         sourceInspect.RowStart,
	}

	for i := range imp.Columns {
		col := imp.Columns[i].ToMap()
		name, ok := col["name"].(string)
		if !ok {
			continue
		}
		binding := Binding{Name: "", Axis: "", Col: -1, Rules: []string{}, Field: name, Label: imp.Columns[i].Label}
		for _, suggest := range imp.Columns[i].Match {
			if srcCol, has := sourceColumns[suggest]; has {
				binding = Binding{
					Label: imp.Columns[i].Label,
					Field: name,
					Name:  srcCol.Name,
					Axis:  srcCol.Axis,
					Col:   srcCol.Col,
					Rules: imp.Columns[i].Rules,
				}
				continue
			}
		}
		mapping.Columns = append(mapping.Columns, &binding)
	}

	return mapping
}

// DataGet 读取源数据记录
func (imp *Importer) DataGet(src from.Source, page int, size int, mapping *Mapping) ([]string, [][]interface{}) {

	row := (page-1)*size + mapping.RowStart
	if row < 0 {
		row = mapping.RowStart
	}
	cols := []int{}
	for _, d := range mapping.Columns {
		cols = append(cols, d.Col)
	}
	data := src.Data(row, size, cols)
	return imp.DataClean(data, mapping.Columns)
}

// Chunk 遍历数据
func (imp *Importer) Chunk(src from.Source, mapping *Mapping, cb func(line int, data [][]interface{})) {
	cols := []int{}
	for _, d := range mapping.Columns {
		cols = append(cols, d.Col)
	}
	src.Chunk(imp.Option.ChunkSize, cols, cb)
}

// DataClean 清洗数据
func (imp *Importer) DataClean(data [][]interface{}, bindings []*Binding) ([]string, [][]interface{}) {
	columns := []string{}
	new := [][]interface{}{}

	for _, binding := range bindings {
		columns = append(columns, binding.Field)
	}
	// 清洗数据
	for _, row := range data {
		success := true
		for i, binding := range bindings { // 调用字段清洗处理器
			for _, rule := range binding.Rules {
				if !DataValidate(row, row[i], rule) {
					success = false
				}
			}
		}
		row = append(row, success)
		new = append(new, row)
	}

	columns = append(columns, "__effected")
	return columns, new
}

// DataValidate 数值校验
func DataValidate(row []interface{}, value interface{}, rule string) bool {
	process, err := gou.ProcessOf(rule, value, row)
	if err != nil {
		xlog.Printf("DataValidate: %s %s", rule, err.Error())
		return true
	}
	res, err := process.Exec()
	if err != nil {
		xlog.Printf("DataValidate: %s %s", rule, err.Error())
		return true
	}

	if update, ok := res.([]interface{}); ok {
		row = update
		return true
	}
	return false
}

// DataPreview 预览数据
func (imp *Importer) DataPreview(src from.Source, page int, size int, mapping *Mapping) map[string]interface{} {
	if page < 1 {
		page = 1
	}

	data := []map[string]interface{}{}
	res := map[string]interface{}{
		"page":     page,
		"pagesize": size,
		"pagecnt":  10,
		"next":     page + 1,
		"prev":     page - 1,
	}

	if mapping == nil {
		mapping = imp.AutoMapping(src)
	}

	columns, rows := imp.DataGet(src, page, size, mapping)
	for _, row := range rows {
		if len(row) != len(columns) {
			exception.New("数据异常, 请联系管理员", 500).Ctx(map[string]interface{}{"row": row, "columns": columns}).Throw()
		}

		rs := map[string]interface{}{}
		for i := range row {
			key := columns[i]
			value := row[i]
			rs[key] = value
		}

		data = append(data, rs)
	}

	res["data"] = data
	return res
}

// MappingPreview 预览字段映射关系
func (imp *Importer) MappingPreview(src from.Source) *Mapping {

	// 模板匹配(下一版实现)
	// tpl := imp.Fingerprint(src)
	// 查找已有模板

	// 自动匹配
	return imp.AutoMapping(src)
}

// DataSetting 预览数据表格配置
func (imp *Importer) DataSetting(src from.Source) []map[string]interface{} {
	return nil
}

// MappingSetting 预览映射数据表格配置
func (imp *Importer) MappingSetting(src from.Source) []map[string]interface{} {
	return nil
}

// Fingerprint 文件结构指纹
func (imp *Importer) Fingerprint(src from.Source) string {
	keys := []string{}
	columns := src.Columns()
	for _, col := range columns {
		keys = append(keys, fmt.Sprintf("%s|%d", col.Name, col.Type))
	}
	sort.Strings(keys)
	hash := sha256.New()
	hash.Write([]byte(strings.Join(keys, "")))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// SaveAsTemplate 保存为映射模板
func (imp *Importer) SaveAsTemplate(src from.Source) {
}

// Run 运行导入
func (imp *Importer) Run(src from.Source, mapping *Mapping) map[string]int {
	if mapping == nil {
		mapping = imp.AutoMapping(src)
	}

	total := 0
	failed := 0
	ignore := 0
	imp.Chunk(src, mapping, func(line int, data [][]interface{}) {
		length := len(data)
		total = total + length
		columns, data := imp.DataClean(data, mapping.Columns)
		process, err := gou.ProcessOf(imp.Process, columns, data)
		if err != nil {
			failed = failed + length
			xlog.Printf("导入失败 %d %s ", line, err.Error())
			return
		}

		response, err := process.Exec()
		if err != nil {
			failed = failed + length
			xlog.Printf("导入失败 %d %s ", line, err.Error())
			return
		}

		if res, ok := response.([]int); ok && len(res) > 1 {
			failed = failed + res[0]
			ignore = ignore + res[1]
			return
		} else if res, ok := response.([]int64); ok && len(res) > 1 {
			failed = failed + int(res[0])
			ignore = ignore + int(res[1])
			return
		} else if res, ok := response.([]interface{}); ok && len(res) > 1 {
			failed = failed + any.Of(res[0]).CInt()
			ignore = ignore + any.Of(res[1]).CInt()
			return
		}

		xlog.Printf("导入处理器未返回失败结果 %#v %d %d", response, line, length)
	})
	return map[string]int{
		"total":   total,
		"success": total - failed - ignore,
		"failure": failed,
		"ignore":  ignore,
	}
}

// Start 运行导入(异步)
func (imp *Importer) Start() {}

// getSourceColumns 读取源数据字段映射表
func getSourceColumns(src from.Source) map[string]from.Column {
	res := map[string]from.Column{}
	columns := src.Columns()
	for _, col := range columns {
		name := col.Name
		if name != "" {
			res[name] = col
		}
	}
	return res
}

// getColumns 读取目标字段映射表
func (imp *Importer) getColumns() map[string]*Column {
	columns := map[string]*Column{}
	for i := range imp.Columns {
		colmap := imp.Columns[i].ToMap()

		if name, ok := colmap["name"].(string); ok && name != "" {
			columns[name] = &imp.Columns[i]
		}
	}
	return columns
}
