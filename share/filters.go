package share

import (
	"fmt"

	"github.com/yaoapp/gou/model"
)

// GetDefaultFilters 读取数据模型索引字段的过滤器
func GetDefaultFilters(name string) map[string]Filter {

	mod := model.Select(name)
	cmap := mod.Columns
	filters := map[string]Filter{}
	for _, index := range mod.MetaData.Indexes {
		for _, col := range index.Columns {
			if _, has := cmap[col]; !has {
				continue
			}
			// primary,unique,index,match
			switch index.Type {
			case "index", "match":
				cmap[col].Index = true
				break
			case "unique":
				if len(index.Columns) == 1 {
					cmap[col].Unique = true
				}
				break
			case "primary":
				cmap[col].Primary = true
				break
			}
		}
	}

	for name, col := range cmap {

		if col.Type != "ID" && !col.Index && !col.Unique && !col.Primary {
			continue
		}

		vcol, has := elms[col.Type]
		if !has {
			continue
		}

		label := col.Label
		if label == "" {
			label = col.Comment
		}
		if label == "" {
			label = name
		}

		filter := Filter{
			Label: label,
			Bind:  fmt.Sprintf("where.%s.eq", name),
			Input: vcol.Edit,
		}
		filters[name] = filter
		filters[label] = filter
	}

	return filters

}
