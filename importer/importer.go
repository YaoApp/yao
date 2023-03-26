package importer

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/importer/from"
	"github.com/yaoapp/yao/importer/xlsx"
	"github.com/yaoapp/yao/share"
)

// Importers 导入器
var Importers = map[string]*Importer{}

// DataRoot data file root
var DataRoot string = ""

// Load 加载导入器
func Load(cfg config.Config) error {

	fs, err := fs.Get("system")
	if err != nil {
		return err
	}

	DataRoot = fs.Root()

	exts := []string{"*.imp.yao", "*.imp.json", "*.imp.jsonc"}
	return application.App.Walk("imports", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := share.ID(root, file)
		data, err := application.App.Read(file)
		if err != nil {
			return err
		}

		var importer Importer
		err = application.Parse(file, data, &importer)
		if err != nil {
			return fmt.Errorf("%s 导入配置错误. %s", id, err.Error())
		}

		Importers[id] = &importer
		return nil
	}, exts...)
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
		file := filepath.Join(DataRoot, name)
		return xlsx.Open(file)
	}
	exception.New("暂不支持: %s 文件导入", 400, ext).Throw()
	return nil
}

// WithSid attch sid
func (imp *Importer) WithSid(sid string) *Importer {
	imp.Sid = sid
	return imp
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
		binding := Binding{Name: "", Axis: "", Rules: []string{}, Field: name, Label: imp.Columns[i].Label}
		for _, suggest := range imp.Columns[i].Match {
			if srcCol, has := sourceColumns[suggest]; has {
				binding = Binding{
					Label: imp.Columns[i].Label,
					Field: name,
					Name:  srcCol.Name,
					Axis:  srcCol.Axis,
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
	axises := []string{}
	for _, d := range mapping.Columns {
		axises = append(axises, d.Axis)
	}
	data := src.Data(row, size, axises)
	return imp.DataClean(data, mapping.Columns)
}

// Chunk 遍历数据
func (imp *Importer) Chunk(src from.Source, mapping *Mapping, cb func(line int, data [][]interface{})) {
	axises := []string{}
	for _, d := range mapping.Columns {
		axises = append(axises, d.Axis)
	}
	src.Chunk(imp.Option.ChunkSize, axises, cb)
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
				update, ok := DataValidate(row, row[i], rule)
				if !ok {
					success = false
				} else {
					row = update
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
func DataValidate(row []interface{}, value interface{}, rule string) ([]interface{}, bool) {
	process, err := process.Of(rule, value, row)
	if err != nil {
		log.With(log.F{"rule": rule, "row": row}).Error("DataValidate: %s", err.Error())
		return row, true
	}
	res, err := process.Exec()
	if err != nil {
		log.With(log.F{"rule": rule, "row": row}).Error("DataValidate: %s", err.Error())
		return row, true
	}

	if update, ok := res.([]interface{}); ok {
		row = update
		return update, true
	}
	return row, false
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
	for idx, row := range rows {
		if len(row) != len(columns) {
			exception.New("数据异常, 请联系管理员", 500).Ctx(map[string]interface{}{"row": row, "columns": columns}).Throw()
		}
		rs := map[string]interface{}{}
		for i := range row {
			key := columns[i]
			value := row[i]
			rs[key] = value
		}
		rs["id"] = idx + 1
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

	mapping := imp.AutoMapping(src) // 自动匹配

	// 预设值
	columns, rows := imp.DataGet(src, 1, 1, mapping)
	if len(rows) > 0 {
		row := rows[0]
		rs := map[string]interface{}{}
		for i := range row {
			key := columns[i]
			value := row[i]
			rs[key] = value
		}
		for i := range mapping.Columns {
			name := mapping.Columns[i].Field
			mapping.Columns[i].Value = fmt.Sprintf("%v", rs[name])
		}
	}
	return mapping
}

// DataSetting 预览数据表格配置
func (imp *Importer) DataSetting() map[string]interface{} {

	columns := map[string]share.Column{}
	layoutColumns := []map[string]interface{}{}
	for _, column := range imp.Columns {
		name := column.Label
		layoutColumns = append(layoutColumns, map[string]interface{}{"name": name})
		columns[name] = share.Column{
			Label: name,
			View: share.Render{
				Type:  "label",
				Props: map[string]interface{}{"value": fmt.Sprintf(":%s", column.Field)},
			},
		}
	}

	setting := map[string]interface{}{
		"columns": columns,
		"filters": map[string]interface{}{},
		"list": share.Page{
			Primary: "id",
			Layout:  map[string]interface{}{"columns": layoutColumns},
			Actions: map[string]share.Render{
				"pagination": {
					Props: map[string]interface{}{
						"showTotal": true,
					},
				},
			},
			Option: map[string]interface{}{
				"operation": map[string]interface{}{
					"hideView": true,
					"hideEdit": true,
					"width":    120,
					"unfold":   true,
					"checkbox": []map[string]interface{}{{
						"value":         ":__effected",
						"visible_label": false,
						"status": []map[string]interface{}{
							{
								"label": "有效",
								"value": true,
							},
							{
								"label": "无效",
								"value": false,
							},
						},
					}},
				},
			},
		},
	}
	return setting
}

// MappingSetting 预览映射数据表格配置
func (imp *Importer) MappingSetting(src from.Source) map[string]interface{} {

	columns := map[string]share.Column{
		"字段名称": {
			Label: "字段名称",
			View: share.Render{
				Type:  "label",
				Props: map[string]interface{}{"value": ":label"},
			},
		},
		"数据源": {
			Label: "数据源",
			View: share.Render{
				Type:  "label",
				Props: map[string]interface{}{"value": ":name"},
			},
			Edit: share.Render{
				Type:  "select",
				Props: map[string]interface{}{"options": imp.getSourceOption(src), "value": ":axis"},
			},
		},
		"清洗规则": {
			Label: "清洗规则",
			View: share.Render{
				Type:  "tag",
				Props: map[string]interface{}{"value": ":rules"},
			},
			Edit: share.Render{
				Type:  "select",
				Props: map[string]interface{}{"options": imp.getRulesOption(), "value": ":rules", "mode": "multiple"},
			},
		},
		"数据示例": {
			Label: "数据示例",
			View: share.Render{
				Type:  "label",
				Props: map[string]interface{}{"value": ":value"},
			},
		},
	}
	setting := map[string]interface{}{
		"columns": columns,
		"filters": map[string]interface{}{},
		"list": share.Page{
			Primary: "field",
			Layout: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "字段名称"},
					{"name": "数据源"},
					{"name": "清洗规则", "width": 300},
					{"name": "数据示例"},
				},
			},
			Option: map[string]interface{}{
				"operation": map[string]interface{}{"hideView": true, "hideEdit": true, "width": 0},
			},
			Actions: map[string]share.Render{},
		},
	}
	return setting
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
func (imp *Importer) Run(src from.Source, mapping *Mapping) interface{} {
	if mapping == nil {
		mapping = imp.AutoMapping(src)
	}

	id := uuid.NewString()
	page := 0
	total := 0
	failed := 0
	ignore := 0
	imp.Chunk(src, mapping, func(line int, data [][]interface{}) {
		page++
		length := len(data)
		total = total + length
		columns, data := imp.DataClean(data, mapping.Columns)
		process, err := process.Of(imp.Process, columns, data, id, page)
		if err != nil {
			failed = failed + length
			log.With(log.F{"line": line}).Error("导入失败: %s", err.Error())
			return
		}

		response, err := process.WithSID(imp.Sid).Exec()
		if err != nil {
			failed = failed + length
			log.With(log.F{"line": line}).Error("导入失败: %s", err.Error())
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

		log.With(log.F{"line": line, "response": response, "length": length}).Error("导入处理器未返回失败结果")
	})

	output := map[string]int{
		"total":   total,
		"success": total - failed - ignore,
		"failure": failed,
		"ignore":  ignore,
	}

	if imp.Output != "" {
		res, err := process.New(imp.Output, output).WithSID(imp.Sid).Exec()
		if err != nil {
			log.With(log.F{"output": imp.Output}).Error(err.Error())
			return output
		}
		return res
	}

	return output
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

func (imp *Importer) getFieldOption() []map[string]interface{} {
	option := []map[string]interface{}{}
	for _, col := range imp.Columns {
		option = append(option, map[string]interface{}{
			"label": col.Label, "value": col.Field,
		})
	}
	return option
}

func (imp *Importer) getSourceOption(src from.Source) []map[string]interface{} {
	option := []map[string]interface{}{}
	columns := src.Columns()
	for _, col := range columns {
		option = append(option, map[string]interface{}{
			"label": col.Name, "value": col.Axis,
		})
	}
	return option
}

func (imp *Importer) getRulesOption() []map[string]interface{} {
	option := []map[string]interface{}{}
	keys := []string{}
	for key := range imp.Rules {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		option = append(option, map[string]interface{}{
			"label": imp.Rules[key], "value": key,
		})
	}
	return option
}
