package component

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/maps"
)

func TestComputeUnmarshalJSON(t *testing.T) {

	tests := testComputeData()

	var compute Compute
	err := jsoniter.Unmarshal(tests["Trim"], &compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Trim", compute.Process)
	assert.Equal(t, true, compute.Args[0].IsExp)
	assert.Equal(t, "value", compute.Args[0].key)
	assert.Equal(t, nil, compute.Args[0].value)
	assert.Equal(t, true, compute.Args[1].IsExp)
	assert.Equal(t, "props", compute.Args[1].key)
	assert.Equal(t, nil, compute.Args[1].value)
	assert.Equal(t, true, compute.Args[2].IsExp)
	assert.Equal(t, "type", compute.Args[2].key)
	assert.Equal(t, nil, compute.Args[2].value)

	err = jsoniter.Unmarshal(tests["Concat"], &compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Concat", compute.Process)
	assert.Equal(t, false, compute.Args[1].IsExp)
	assert.Equal(t, "::", compute.Args[1].value)
	assert.Equal(t, "", compute.Args[1].key)

	err = jsoniter.Unmarshal(tests["Mapping"], &compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Mapping", compute.Process)
	assert.Equal(t, false, compute.Args[1].IsExp)
	assert.Equal(t, "checked", compute.Args[1].value.(map[string]interface{})["0"])
	assert.Equal(t, "", compute.Args[1].key)

	err = jsoniter.Unmarshal(tests["MappingOnline"], &compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "scripts.compute.MappingOnline", compute.Process)

	err = jsoniter.Unmarshal(tests["Empty"], &compute)
	assert.NotNil(t, err)
	assert.Equal(t, "", compute.Process)

	err = jsoniter.Unmarshal(tests["Error"], &compute)
	assert.NotNil(t, err)
	assert.Equal(t, "", compute.Process)
}

func TestComputeMarshalJSON(t *testing.T) {

	tests := testComputeData()

	var compute Compute
	err := jsoniter.Unmarshal(tests["Trim"], &compute)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := jsoniter.Marshal(compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, tests["Trim"], bytes)

	err = jsoniter.Unmarshal(tests["Concat"], &compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Concat", compute.Process)
	bytes, err = jsoniter.Marshal(compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(bytes), `Concat`)
	assert.Contains(t, string(bytes), `$C(value)`)
	assert.Contains(t, string(bytes), `\\::`)

	err = jsoniter.Unmarshal(tests["Mapping"], &compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Mapping", compute.Process)
	bytes, err = jsoniter.Marshal(compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(bytes), `Mapping`)
	assert.Contains(t, string(bytes), `curing`)

	err = jsoniter.Unmarshal(tests["Empty"], &compute)
	assert.NotNil(t, err)
	assert.Equal(t, "", compute.Process)
	bytes, err = jsoniter.Marshal(compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, `""`, string(bytes))

	err = jsoniter.Unmarshal(tests["Error"], &compute)
	assert.NotNil(t, err)
	assert.Equal(t, "", compute.Process)
	bytes, err = jsoniter.Marshal(compute)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, `""`, string(bytes))

}

func TestComputeValue(t *testing.T) {
	tests := testComputeData()

	data := maps.MapStr{
		"value":      " Concat-Test ",
		"row.type":   "UnitTest",
		"row.status": "enabled",
	}

	var compute Compute
	err := jsoniter.Unmarshal(tests["Concat"], &compute)
	if err != nil {
		t.Fatal(err)
	}

	id := session.ID()
	res, err := compute.Value(data, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "UnitTest:: Concat-Test -enabled", res)

	err = jsoniter.Unmarshal(tests["Trim"], &compute)
	if err != nil {
		t.Fatal(err)
	}

	res, err = compute.Value(data, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Concat-Test", res)

	compute = Compute{Process: "NotFound", Args: []CArg{}}
	res, err = compute.Value(data, id, nil)
	assert.Contains(t, err.Error(), "does not found")
	assert.Nil(t, res)
}

func testComputeData() map[string][]byte {

	return map[string][]byte{
		"Trim": []byte(`"Trim"`),

		"Concat": []byte(`{
            "process": "Concat",
            "args": ["$C(row.type)", "\\::", "$C(value)", "-", "$C(row.status)"]
        }`),

		"Mapping": []byte(` {
            "process": "Mapping",
            "args": [
              "$C(value)",
              { "0": "checked", "1": "curing", "2": "cured" }
            ]
        }`),

		"MappingOnline": []byte(`{
            "process": "scripts.compute.MappingOnline",
            "args": ["$C(value)", "$C(props.mapping)"]
        }`),

		"Empty": []byte(`""`),
		"Error": []byte("[]"),
	}
}
