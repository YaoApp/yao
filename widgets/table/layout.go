package table

import (
	"github.com/yaoapp/gou"
)

// BindModel bind model
func (layout *LayoutDSL) BindModel(m *gou.Model) {
	layout.Primary = m.PrimaryKey
}
