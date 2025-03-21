package excel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEachCols(t *testing.T) {
	files := testFiles(t)
	h1, err := Open(files["test-01"], false)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(h1)

	xls, err := Get(h1)
	if err != nil {
		t.Fatal(err)
	}

	id, err := xls.OpenColumn("供销存管理表格")
	if err != nil {
		t.Fatal(err)
	}
	defer CloseColumn(id)

	res := []string{}
	for col, err := NextColumn(id); err == nil && col != nil; col, err = NextColumn(id) {
		res = append(res, col...)
	}

	assert.Contains(t, strings.Join(res, ""), "供销存管理表格产品查询")
	assert.Contains(t, strings.Join(res, ""), "刘大大")
}

func TestEachRows(t *testing.T) {
	files := testFiles(t)
	h1, err := Open(files["test-01"], false)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(h1)

	xls, err := Get(h1)
	if err != nil {
		t.Fatal(err)
	}

	id, err := xls.OpenRow("供销存管理表格")
	if err != nil {
		t.Fatal(err)
	}
	defer CloseRow(id)

	res := []string{}
	for row, err := NextRow(id); err == nil && row != nil; row, err = NextRow(id) {
		res = append(res, row...)
	}

	assert.Contains(t, strings.Join(res, ""), "供销存管理表格产品查询")
	assert.Contains(t, strings.Join(res, ""), "刘大大")
}
