package field

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/widgets/component"
)

func TestColumnReplace(t *testing.T) {
	prepare(t)

	col := testColumn()
	data := testData()
	new, err := col.Replace(data)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Bar", new.Key)
	assert.Equal(t, "Foo", new.Bind)
	assert.Equal(t, "::please select Bar", new.Edit.Props["placeholder"])
	assert.Equal(t, "::Hello", new.Edit.Props["options"].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", new.Edit.Props["options"].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", new.Edit.Props["options"].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", new.Edit.Props["options"].([]map[string]interface{})[1]["value"])
}

func testColumn() ColumnDSL {
	return ColumnDSL{
		Key:  "${label || comment}",
		Bind: "${name}",
		View: &component.DSL{
			Type:  "Tag",
			Props: component.PropsDSL{"pure": true},
		},
		Edit: &component.DSL{
			Type: "Select",
			Props: component.PropsDSL{
				"placeholder": "::please select ${label || comment}",
				"options":     "$.SelectOption{option}",
			},
		},
	}
}
