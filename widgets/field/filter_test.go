package field

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/widgets/component"
)

func TestFilterReplace(t *testing.T) {
	prepare(t)

	filter := testFilter()
	data := testData()
	new, err := filter.Replace(data)
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

func testFilter() FilterDSL {
	return FilterDSL{
		Key:  "${label || comment}",
		Bind: "${name}",
		Edit: &component.DSL{
			Type: "Select",
			Props: component.PropsDSL{
				"placeholder": "::please select ${label || comment}",
				"options":     "$.SelectOption{option}",
			},
		},
	}
}
