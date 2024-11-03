package component

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessGetOptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	props := prepare(t)

	name := "yao.component.GetOptions"
	for _, queryParam := range props {

		args := []interface{}{
			map[string]interface{}{},
			map[string]interface{}{"query": queryParam},
		}

		p, err := process.Of(name, args...)
		if err != nil {
			t.Fatal(err)
		}

		err = p.Execute()
		if err != nil {
			t.Fatal(err)
		}
		defer p.Release()
		res, ok := p.Value().([]Option)
		if !ok {
			t.Fatal("Result is not []Option")
		}

		if len(res) != 8 {
			t.Fatal("Result length is not 8")
		}

		assert.Equal(t, "Category cat 1-active-1", res[0].Label)
		assert.Equal(t, "1", res[0].Value)
		assert.Equal(t, "active-1", res[0].Icon)

		// With keywords
		args = []interface{}{
			map[string]interface{}{"keywords": "dog"},
			map[string]interface{}{"query": queryParam},
		}

		p, err = process.Of(name, args...)
		if err != nil {
			t.Fatal(err)
		}

		err = p.Execute()
		if err != nil {
			t.Fatal(err)
		}

		res, ok = p.Value().([]Option)
		if !ok {
			t.Fatal("Result is not []Option")
		}

		if len(res) != 2 {
			t.Fatal("Result length is not 2")
		}

		assert.Equal(t, "Category dog 7-active-7", res[0].Label)
		assert.Equal(t, "7", res[0].Value)
		assert.Equal(t, "active-7", res[0].Icon)

		// With selected
		args = []interface{}{
			map[string]interface{}{"selected": []interface{}{1}},
			map[string]interface{}{"query": queryParam},
		}

		p, err = process.Of(name, args...)
		if err != nil {
			t.Fatal(err)
		}

		err = p.Execute()
		if err != nil {
			t.Fatal(err)
		}
		res, ok = p.Value().([]Option)
		if !ok {
			t.Fatal("Result is not []Option")
		}

		if len(res) != 1 {
			t.Fatal("Result length is not 1")
		}

		assert.Equal(t, "Category cat 1-active-1", res[0].Label)
		assert.Equal(t, "1", res[0].Value)
		assert.Equal(t, "active-1", res[0].Icon)

		// With keywords and selected
		args = []interface{}{
			map[string]interface{}{"keywords": "dog", "selected": []interface{}{1, 2}},
			map[string]interface{}{"query": queryParam},
		}

		p, err = process.Of(name, args...)
		if err != nil {
			t.Fatal(err)
		}

		err = p.Execute()
		if err != nil {
			t.Fatal(err)
		}

		res, ok = p.Value().([]Option)
		if !ok {
			t.Fatal("Result is not []Option")
		}

		if len(res) != 4 {
			t.Fatal("Result length is not 4")
		}

		assert.Equal(t, "Category cat 1-active-1", res[0].Label)
		assert.Equal(t, "1", res[0].Value)
		assert.Equal(t, "active-1", res[0].Icon)
		assert.Equal(t, "Category dog 7-active-7", res[2].Label)
		assert.Equal(t, "7", res[2].Value)
		assert.Equal(t, "active-7", res[2].Icon)
	}
}

func TestProcessSelectOptions(t *testing.T) {

	name := "yao.component.SelectOptions"

	p, err := process.Of(name, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = p.Execute()
	assert.Contains(t, err.Error(), "process yao.component.SelectOptions is deprecated, please use yao.component.GetOptions instead")
}

func prepare(t *testing.T) map[string]map[string]interface{} {
	exportProcess()

	// Prepare data for testing
	err := process.New("models.category.Migrate", true).Execute()
	if err != nil {
		t.Fatal(err)
	}

	err = process.New("models.category.Insert",
		[]string{"name", "status"},
		[][]interface{}{
			{"Category cat 1", "active-1"},
			{"Category cat 2", "active-2"},
			{"Category cat 3", "active-3"},
			{"Category cat 4", "active-4"},
			{"Category cat 5", "active-5"},
			{"Category cat 6", "active-6"},
			{"Category dog 7", "active-7"},
			{"Category dog 8", "active-8"},
		}).Execute()
	if err != nil {
		t.Fatal(err)
	}

	queryParam := map[string]interface{}{}
	queryDSL := map[string]interface{}{}
	err = jsoniter.Unmarshal([]byte(`{
		"labelField": "name",
		"valueField": "id",
		"iconField": "status",
		"from": "category",
		"wheres": [
			{ "column": "name", "value": "[[ $keywords ]]", "op": "match" },
			{
				"method": "orwhere",
				"column": "id",
				"op": "in",
				"value": "[[ $selected ]]"
			}
		],
		"limit": 20,
	  	"labelFormat": "[[ $name ]]-[[ $status ]]", 
        "valueFormat": "[[ $id ]]",
        "iconFormat": "[[ $status ]]" 
   }`), &queryParam)

	if err != nil {
		t.Fatal(err)
	}

	err = jsoniter.Unmarshal([]byte(`{
	  	"engine": "query-test", 
        "select": ["name as label", "id as value", "status as icon"],
        "from": "category",
        "wheres": [
          { "field": "name", "match": "[[ $keywords ]]" },
          { "or": true, "field":"id", "in":"[[ $selected ]]" }
        ],
        "limit": 20,
       	"labelFormat": "[[ $label ]]-[[ $icon ]]", 
        "valueFormat": "[[ $value ]]",
        "iconFormat": "[[ $icon ]]" 
	}`), &queryDSL)
	if err != nil {
		t.Fatal(err)
	}

	return map[string]map[string]interface{}{
		"queryParam": queryParam,
		"queryDSL":   queryDSL,
	}
}
