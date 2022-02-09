package share

import (
	"os"
	"path"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func init() {
	rootLib := path.Join(os.Getenv("YAO_DEV"), "/tests/libs")
	LoadFrom(rootLib)
}

func TestColumn(t *testing.T) {
	content := `{ "@": "column.Image", "in": ["LOGO", ":logo", 40] }`
	column := Column{}
	jsoniter.Unmarshal([]byte(content), &column)
	assert.Equal(t, "upload", column.Edit.Type)
	assert.Equal(t, ":logo", column.Edit.Props["value"])
	assert.Equal(t, "image", column.View.Type)
	assert.Equal(t, float64(40), column.View.Props["height"])
	assert.Equal(t, float64(40), column.View.Props["width"])
	assert.Equal(t, ":logo", column.View.Props["value"])
}

func TestColumnInIsNil(t *testing.T) {
	content := `{ "@": "column.创建时间" }`
	column := Column{}
	jsoniter.Unmarshal([]byte(content), &column)
	assert.Equal(t, ":created_at", column.View.Props["value"])
	assert.Equal(t, "创建时间", column.Label)
}

func TestFilter(t *testing.T) {
	content := `{ "@": "filter.关键词", "in": ["where.name.match"] }`
	filter := Filter{}
	jsoniter.Unmarshal([]byte(content), &filter)
	assert.Equal(t, "where.name.match", filter.Bind)
}

func TestRender(t *testing.T) {
	content := `{ "@": "render.Image", "in": [":image", 40, 60] }`
	render := Render{}
	jsoniter.Unmarshal([]byte(content), &render)
	assert.Equal(t, ":image", render.Props["value"])
	assert.Equal(t, float64(40), render.Props["width"])
	assert.Equal(t, float64(60), render.Props["height"])
}

func TestPage(t *testing.T) {
	content := `{ "@": "pages.static.Page", "in": ["id"] }`
	page := Page{}
	jsoniter.Unmarshal([]byte(content), &page)
	assert.Equal(t, "id", page.Primary)
}

func TestAPI(t *testing.T) {
	content := `{ "@": "apis.table.Search", "in": [10] }`
	api := API{}
	jsoniter.Unmarshal([]byte(content), &api)
	assert.Equal(t, []interface{}{nil, nil, float64(10)}, api.Default)
}
