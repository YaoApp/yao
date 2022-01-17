package importer

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/importer/from"
	"github.com/yaoapp/xiang/share"
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

// MappingPreview 预览字段映射关系
func (imp *Importer) MappingPreview(src from.Source) *Mapping {

	// 读取文件结构指纹
	// tpl := imp.Fingerprint(src)
	// 查找已有模板

	// 自动匹配
	imp.AutoMapping(src)

	return nil
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
func (imp *Importer) Run() {}

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
