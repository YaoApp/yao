package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
)

type TestMap map[string]interface{}
type TestSlice []interface{}
type TestStruct struct {
	Name   string
	Map    TestMap
	Slice  TestSlice
	Option TestOption
}

type TestOption struct {
	Int   int
	Float float32
	Bool  bool
	Map   TestMap
	Slice TestSlice
	Nest  TestNest
}

type TestNest struct {
	Int   int
	Float float32
	Bool  bool
	Map   TestMap
	Slice TestSlice
}

func TestReplaceString(t *testing.T) {

	prepare(t)

	data := testData()
	err := Replace(nil, data)
	assert.NotNil(t, err)

	err = Replace(0.618, data)
	assert.NotNil(t, err)

	err = Replace(1, data)
	assert.NotNil(t, err)

	err = Replace("${label}", data)
	assert.NotNil(t, err)

	strv := ""
	err = Replace(&strv, data)
	assert.Equal(t, "", strv)

	strv = "hello world"
	err = Replace(&strv, data)
	assert.Equal(t, "hello world", strv)

	strv = "hello world \\${name}"
	err = Replace(&strv, data)
	assert.Equal(t, "hello world \\${name}", strv)

	strv = "\\$.SelectOption{option}"
	err = Replace(&strv, data)
	assert.Equal(t, "\\$.SelectOption{option}", strv)

	strv = "${name}"
	err = Replace(&strv, data)
	assert.Equal(t, "Foo", strv)

	strv = "${ name }"
	err = Replace(&strv, data)
	assert.Equal(t, "Foo", strv)

	strv = "${ name } and ${label}"
	err = Replace(&strv, data)
	assert.Equal(t, "Foo and Bar", strv)

	strv = "please select ${ name }"
	err = Replace(&strv, data)
	assert.Equal(t, "please select Foo", strv)

	strv = "${label || comment}"
	err = Replace(&strv, data)
	assert.Equal(t, "Bar", strv)

	strv = "${ comment || label }"
	err = Replace(&strv, data)
	assert.Equal(t, "Hi", strv)

	strv = "${name || 'value' || 0.618 || 1}"
	err = Replace(&strv, data)
	assert.Equal(t, "Foo", strv)

	strv = "${ 'value' || name || 0.618 || 1}"
	err = Replace(&strv, data)
	assert.Equal(t, "value", strv)

	strv = "${ 0.618 || 'value' || name || 1}"
	err = Replace(&strv, data)
	assert.Equal(t, "0.618", strv)

	strv = "${ 1 || 0.618 || 'value' || name }"
	err = Replace(&strv, data)
	assert.Equal(t, "1", strv)

	strv = "please select ${ label || comment }"
	err = Replace(&strv, data)
	assert.Equal(t, "please select Bar", strv)

	strv = "please select ${ comment || label  }"
	err = Replace(&strv, data)
	assert.Equal(t, "please select Hi", strv)

	strv = "$.TrimSpace{ space }"
	err = Replace(&strv, data)
	assert.Equal(t, "Hello World", strv)

	intv := 1024
	err = Replace(&intv, data)
	assert.Equal(t, 1024, intv)

	floatv := 0.168
	err = Replace(&floatv, data)
	assert.Equal(t, 0.168, floatv)

}

func TestReplaceMap(t *testing.T) {
	prepare(t)
	data := testData()
	mapv := testMap()
	err := Replace(&mapv, data)
	assert.Nil(t, err)
	assert.Equal(t, "::please select Bar", mapv["placeholder"])
	assert.Equal(t, "::Hello", mapv["options"].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", mapv["options"].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", mapv["options"].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", mapv["options"].([]map[string]interface{})[1]["value"])
}

func TestReplaceSlice(t *testing.T) {
	prepare(t)
	data := testData()
	arrv := testSlice()
	err := Replace(&arrv, data)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(arrv))
	assert.Equal(t, "::please select Bar", arrv[0])
	assert.Equal(t, "::Hello", arrv[1].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", arrv[1].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", arrv[1].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", arrv[1].([]map[string]interface{})[1]["value"])
}

func TestReplaceNest(t *testing.T) {
	prepare(t)
	data := testData()
	nestv := testNest()
	err := Replace(&nestv, data)
	assert.Nil(t, err)
	assert.Equal(t, "::please select Bar", nestv["placeholder"])
	assert.Equal(t, "::Hello", nestv["options"].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", nestv["options"].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", nestv["options"].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", nestv["options"].([]map[string]interface{})[1]["value"])

	arrv := nestv["data"].([]interface{})
	assert.Equal(t, 2, len(arrv))
	assert.Equal(t, "::please select Bar", arrv[0])
	assert.Equal(t, "::Hello", arrv[1].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", arrv[1].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", arrv[1].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", arrv[1].([]map[string]interface{})[1]["value"])
}

func TestReplaceStruct(t *testing.T) {
	prepare(t)
	data := testData()
	structv := testStruct()
	err := Replace(&structv, data)
	assert.Nil(t, err)

	assert.Equal(t, "Bar", structv.Name)
	assert.Equal(t, "::Hello", structv.Map["options"].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", structv.Map["options"].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", structv.Map["options"].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", structv.Map["options"].([]map[string]interface{})[1]["value"])

	arrv := structv.Slice
	assert.Equal(t, 2, len(arrv))
	assert.Equal(t, "::please select Bar", arrv[0])
	assert.Equal(t, "::Hello", arrv[1].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", arrv[1].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", arrv[1].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", arrv[1].([]map[string]interface{})[1]["value"])
}

func TestReplaceAny(t *testing.T) {
	prepare(t)

	data := testData()
	var anyv interface{} = ""
	err := Replace(&anyv, data)
	assert.Nil(t, err)
	assert.Equal(t, "", anyv)

	anyv = "hello world"
	err = Replace(&anyv, data)
	assert.Equal(t, "hello world", anyv)

	anyv = "hello world \\${name}"
	err = Replace(&anyv, data)
	assert.Equal(t, "hello world \\${name}", anyv)

	anyv = "\\$.SelectOption{option}"
	err = Replace(&anyv, data)
	assert.Equal(t, "\\$.SelectOption{option}", anyv)

	anyv = "${name}"
	err = Replace(&anyv, data)
	assert.Equal(t, "Foo", anyv)

	anyv = "${ name }"
	err = Replace(&anyv, data)
	assert.Equal(t, "Foo", anyv)

	anyv = "${ name } and ${label}"
	err = Replace(&anyv, data)
	assert.Equal(t, "Foo and Bar", anyv)

	anyv = "please select ${ name }"
	err = Replace(&anyv, data)
	assert.Equal(t, "please select Foo", anyv)

	anyv = "${label || comment}"
	err = Replace(&anyv, data)
	assert.Equal(t, "Bar", anyv)

	anyv = "${ comment || label }"
	err = Replace(&anyv, data)
	assert.Equal(t, "Hi", anyv)

	anyv = "${name || 'value' || 0.618 || 1}"
	err = Replace(&anyv, data)
	assert.Equal(t, "Foo", anyv)

	anyv = "${ 'value' || name || 0.618 || 1}"
	err = Replace(&anyv, data)
	assert.Equal(t, "value", anyv)

	anyv = "${ 0.618 || 'value' || name || 1}"
	err = Replace(&anyv, data)
	assert.Equal(t, "0.618", anyv)

	anyv = "${ 1 || 0.618 || 'value' || name }"
	err = Replace(&anyv, data)
	assert.Equal(t, "1", anyv)

	anyv = "please select ${ label || comment }"
	err = Replace(&anyv, data)
	assert.Equal(t, "please select Bar", anyv)

	anyv = "please select ${ comment || label  }"
	err = Replace(&anyv, data)
	assert.Equal(t, "please select Hi", anyv)

	anyv = "$.TrimSpace{ space }"
	err = Replace(&anyv, data)
	assert.Equal(t, "Hello World", anyv)

	anyv = "$.SelectOption{ option }"
	err = Replace(&anyv, data)
	res, ok := anyv.([]map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "::Hello", res[0]["label"])
	assert.Equal(t, "Hello", res[0]["value"])
	assert.Equal(t, "::World", res[1]["label"])
	assert.Equal(t, "World", res[1]["value"])

	anyv = 1024
	err = Replace(&anyv, data)
	assert.Equal(t, 1024, anyv)

	anyv = 0.168
	err = Replace(&anyv, data)
	assert.Equal(t, 0.168, anyv)

	anyv = testMap()
	err = Replace(&anyv, data)
	assert.Nil(t, err)
	assert.Equal(t, "::please select Bar", anyv.(TestMap)["placeholder"])
	assert.Equal(t, "::Hello", anyv.(TestMap)["options"].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", anyv.(TestMap)["options"].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", anyv.(TestMap)["options"].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", anyv.(TestMap)["options"].([]map[string]interface{})[1]["value"])

	anyv = testSlice()
	err = Replace(&anyv, data)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(anyv.([]interface{})))
	assert.Equal(t, "::please select Bar", anyv.([]interface{})[0])
	assert.Equal(t, "::Hello", anyv.([]interface{})[1].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", anyv.([]interface{})[1].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", anyv.([]interface{})[1].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", anyv.([]interface{})[1].([]map[string]interface{})[1]["value"])

	anyv = testNest()
	err = Replace(&anyv, data)
	assert.Nil(t, err)
	assert.Equal(t, "::please select Bar", anyv.(TestMap)["placeholder"])
	assert.Equal(t, "::Hello", anyv.(TestMap)["options"].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", anyv.(TestMap)["options"].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", anyv.(TestMap)["options"].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", anyv.(TestMap)["options"].([]map[string]interface{})[1]["value"])

	arrv := anyv.(TestMap)["data"].([]interface{})
	assert.Equal(t, 2, len(arrv))
	assert.Equal(t, "::please select Bar", arrv[0])
	assert.Equal(t, "::Hello", arrv[1].([]map[string]interface{})[0]["label"])
	assert.Equal(t, "Hello", arrv[1].([]map[string]interface{})[0]["value"])
	assert.Equal(t, "::World", arrv[1].([]map[string]interface{})[1]["label"])
	assert.Equal(t, "World", arrv[1].([]map[string]interface{})[1]["value"])

}

func prepare(t *testing.T) {
	Export()
}

func testMap() TestMap {
	return TestMap{
		"placeholder": "::please select ${label || comment}",
		"options":     "$.SelectOption{option}",
	}
}

func testSlice() TestSlice {
	return []interface{}{
		"::please select ${label || comment}",
		"$.SelectOption{option}",
	}
}

func testStruct() TestStruct {
	return TestStruct{
		Name:  "${label || comment}",
		Map:   testMap(),
		Slice: testSlice(),
		Option: TestOption{
			Int:   1,
			Float: 0.618,
			Bool:  true,
			Map:   testMap(),
			Slice: testSlice(),
			Nest: TestNest{
				Int:   1,
				Float: 0.618,
				Bool:  true,
				Map:   testMap(),
				Slice: testSlice(),
			},
		},
	}
}

func testNest() TestMap {
	return TestMap{
		"placeholder": "::please select ${label || comment}",
		"options":     "$.SelectOption{option}",
		"data":        testSlice(),
	}
}

func testData() map[string]interface{} {
	return maps.MapStr{
		"name":    "Foo",
		"label":   "Bar",
		"comment": "Hi",
		"space":   " Hello World ",
		"variables": map[string]interface{}{
			"color": TestMap{
				"primary": "#FF0000",
			},
		},
		"option": []interface{}{"Hello", "World"},
	}.Dot()
}
