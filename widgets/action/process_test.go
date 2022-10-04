package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBind(t *testing.T) {
	tests := testData()
	tests["T0"].Bind("yao.unit.T0")
	assert.Equal(t, "yao.unit.T0", tests["T0"].ProcessBind)
}

func TestDefaultMerge(t *testing.T) {
	tests := testData()
	D := testProcessDefaults()
	T0 := tests["T0"]

	T0.DefaultMerge(nil)
	assert.Equal(t, []interface{}{nil, nil, nil}, T0.Default)

	T0.DefaultMerge([]interface{}{D["string"]})
	assert.Equal(t, []interface{}{"hello", nil, nil}, T0.Default)

	T0.DefaultMerge([]interface{}{nil, D["float"]})
	assert.Equal(t, []interface{}{"hello", 0.618, nil}, T0.Default)

	T0.DefaultMerge([]interface{}{nil, nil, D["int"]})
	assert.Equal(t, []interface{}{"hello", 0.618, 49}, T0.Default)

	T0.DefaultMerge([]interface{}{nil, nil, nil, D["int"]})
	assert.Equal(t, []interface{}{"hello", 0.618, 49, 49}, T0.Default)

	T0.DefaultMerge([]interface{}{nil, D["string"], nil})
	assert.Equal(t, []interface{}{"hello", 0.618, 49, 49}, T0.Default)

	T0.DefaultMerge([]interface{}{nil, D["string"], nil}, true)
	assert.Equal(t, []interface{}{"hello", "hello", 49, 49}, T0.Default)

	// T1
	T1 := tests["T1"]
	T1.DefaultMerge(nil)
	assert.Equal(t, []interface{}{nil, nil}, T1.Default)

	T1.DefaultMerge([]interface{}{D["map"], D["slice"]})
	assert.Equal(t, 1.38065, T1.Default[0].(map[string]interface{})["float"])
	assert.Equal(t, 64, T1.Default[0].(map[string]interface{})["int"])
	assert.Equal(t, "foo", T1.Default[0].(map[string]interface{})["string"])
	assert.Equal(t, "world", T1.Default[1].([]interface{})[0])
	assert.Equal(t, 9.10939, T1.Default[1].([]interface{})[1])
	assert.Equal(t, 81, T1.Default[1].([]interface{})[2])

	// overwrite false, deep true
	T1.DefaultMerge([]interface{}{
		D["nest"].(map[string]interface{})["nest-map"],
		D["nest"].(map[string]interface{})["nest-slice"],
	})
	assert.Equal(t, 1.38065, T1.Default[0].(map[string]interface{})["float"])
	assert.Equal(t, 64, T1.Default[0].(map[string]interface{})["int"])
	assert.Equal(t, "foo", T1.Default[0].(map[string]interface{})["string"])
	assert.Contains(t, T1.Default[0], "map")
	assert.Contains(t, T1.Default[0], "slice")
	assert.Equal(t, "world", T1.Default[1].([]interface{})[0])
	assert.Equal(t, 9.10939, T1.Default[1].([]interface{})[1])
	assert.Equal(t, 81, T1.Default[1].([]interface{})[2])
	assert.Contains(t, T1.Default[1].([]interface{})[3], "float")
	assert.Contains(t, T1.Default[1].([]interface{})[4], "bar")

	// T2
	// overwrite true, deep false
	T1.DefaultMerge([]interface{}{D["map"], D["slice"]}, true, false)
	assert.Equal(t, 1.38065, T1.Default[0].(map[string]interface{})["float"])
	assert.Equal(t, 64, T1.Default[0].(map[string]interface{})["int"])
	assert.Equal(t, "foo", T1.Default[0].(map[string]interface{})["string"])
	assert.Equal(t, "world", T1.Default[1].([]interface{})[0])
	assert.Equal(t, 9.10939, T1.Default[1].([]interface{})[1])
	assert.Equal(t, 81, T1.Default[1].([]interface{})[2])

	// overwrite true, deep true
	T1.DefaultMerge([]interface{}{
		D["nest"].(map[string]interface{})["nest-map"],
		D["nest"].(map[string]interface{})["nest-slice"],
	}, true, true)
	assert.Equal(t, 3.1415926, T1.Default[0].(map[string]interface{})["float"])
	assert.Equal(t, 99, T1.Default[0].(map[string]interface{})["int"])
	assert.Equal(t, "bar", T1.Default[0].(map[string]interface{})["string"])
	assert.Contains(t, T1.Default[0], "map")
	assert.Contains(t, T1.Default[0], "slice")
	assert.Equal(t, "bar", T1.Default[1].([]interface{})[0])
	assert.Equal(t, 3.1415926, T1.Default[1].([]interface{})[1])
	assert.Equal(t, 99, T1.Default[1].([]interface{})[2])
	assert.Contains(t, T1.Default[1].([]interface{})[3], "float")
	assert.Contains(t, T1.Default[1].([]interface{})[4], "bar")

	// overwrite false, deep false
	T1.DefaultMerge([]interface{}{
		map[string]interface{}{"string": "foo", "hello": "world"},
		[]interface{}{"foo", nil, nil, nil, nil, "world"},
	}, false, false)
	assert.Equal(t, 3.1415926, T1.Default[0].(map[string]interface{})["float"])
	assert.Equal(t, 99, T1.Default[0].(map[string]interface{})["int"])
	assert.Equal(t, "bar", T1.Default[0].(map[string]interface{})["string"])
	assert.Equal(t, "world", T1.Default[0].(map[string]interface{})["hello"])
	assert.Contains(t, T1.Default[0], "map")
	assert.Contains(t, T1.Default[0], "slice")
	assert.Equal(t, "bar", T1.Default[1].([]interface{})[0])
	assert.Equal(t, 3.1415926, T1.Default[1].([]interface{})[1])
	assert.Equal(t, 99, T1.Default[1].([]interface{})[2])
	assert.Contains(t, T1.Default[1].([]interface{})[3], "float")
	assert.Contains(t, T1.Default[1].([]interface{})[4], "bar")
	assert.Contains(t, T1.Default[1].([]interface{})[5], "world")

}

func testProcessDefaults() map[string]interface{} {

	return map[string]interface{}{
		"string": "hello",
		"float":  0.618,
		"int":    49,
		"map": map[string]interface{}{
			"string": "foo",
			"float":  1.38065,
			"int":    64,
		},
		"slice": []interface{}{
			"world",
			9.10939,
			81,
		},
		"nest": map[string]interface{}{
			"string": "bar",
			"float":  3.1415926,
			"int":    99,
			"slice": []interface{}{
				"bar",
				3.1415926,
				99,
			},
			"nest-slice": []interface{}{
				"bar",
				3.1415926,
				99,
				map[string]interface{}{
					"string": "bar",
					"float":  3.1415926,
					"int":    99,
				},

				[]interface{}{
					"bar",
					3.1415926,
					99,
					map[string]interface{}{
						"string": "bar",
						"float":  3.1415926,
						"int":    99,
					},
				},
			},
			"map": map[string]interface{}{
				"string": "bar",
				"float":  3.1415926,
				"int":    99,
			},
			"nest-map": map[string]interface{}{
				"string": "bar",
				"float":  3.1415926,
				"int":    99,
				"map": map[string]interface{}{
					"string": "bar",
					"float":  3.1415926,
					"int":    99,
				},
				"slice": []interface{}{
					"bar",
					3.1415926,
					99,
					map[string]interface{}{
						"string": "bar",
						"float":  3.1415926,
						"int":    99,
					},
				},
			},
		},
	}
}
