package field

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenTransformReplace(t *testing.T) {

	data := testTransformData(t)
	tr, err := OpenTransform(data, "unit-test")
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, tr.Aliases)
	assert.NotNil(t, tr.Fields)
	assert.NotNil(t, tr.Variables)
	_, has := Transforms["unit-test"]
	assert.True(t, has)
}

func TestTransformFilter(t *testing.T) {
	tr := testDefaultTransform(t)
	data := testData()
	fi, err := tr.Filter("string", data)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Bar", fi.Key)
	assert.Equal(t, "where.Foo.match", fi.Bind)
	assert.Equal(t, "$L(please input) Bar", fi.Edit.Props["placeholder"])

	fi, err = tr.Filter("not-found", data)
	assert.True(t, IsNotFound(err))

	fi, err = tr.Filter("text", data)
	assert.True(t, IsNotFound(err))
}

func TestTransformTable(t *testing.T) {
	tr := testDefaultTransform(t)
	data := testData()
	tab, err := tr.Table("string", data)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Bar", tab.Key)
	assert.Equal(t, "Foo", tab.Bind)
	assert.Equal(t, "$L(please input) Bar", tab.Edit.Props["placeholder"])

	tab, err = tr.Table("not-found", data)
	assert.True(t, IsNotFound(err))
}

func TestTransformForm(t *testing.T) {
	tr := testDefaultTransform(t)
	data := testData()
	form, err := tr.Form("string", data)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Bar", form.Key)
	assert.Equal(t, "Foo", form.Bind)
	assert.Equal(t, "$L(please input) Bar", form.Edit.Props["placeholder"])

	form, err = tr.Form("not-found", data)
	assert.True(t, IsNotFound(err))
}

func testTransformData(t *testing.T) []byte {
	file := filepath.Join(os.Getenv("YAO_DEV"), "yao", "fields", "model.trans.json")
	data, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func testDefaultTransform(t *testing.T) *Transform {
	data := testTransformData(t)
	_, err := OpenTransform(data, "unit-test")
	if err != nil {
		t.Fatal(err)
	}

	if _, has := Transforms["unit-test"]; !has {
		t.Fatal(fmt.Errorf("create unit-test transform error"))
	}
	return Transforms["unit-test"]
}
