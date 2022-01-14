package importer

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

var testDataColumns = map[string][]byte{
	"normal": []byte(`{
		"label": "订单号",
		"name": "order_sn",
		"match": ["订单号", "订单", "order_sn", "id"],
		"rules": ["scripts.rules.order_sn"],
		"primary": true
	}`),
	"object": []byte(`{
		"label": "性别",
		"name": "user.sex",
		"match": "性别",
		"rules": ["scripts.rules.FmtUser"],
		"nullable": true
	}`),
	"array": []byte(`{
		"label": "库存",
		"name": "stock[*]",
		"rules": ["scripts.rules.FmtGoods"]
	}`),
	"arrayObject": []byte(`{
		"label": "商品",
		"name": "skus[*].name",
		"match": ["商品", "商品名称", "goods", "skus", "sku_id", "goods_id"],
		"rules": ["scripts.rules.FmtGoods"]
	}`),
	"failure": []byte(`{
		"xx": "商品",
		"sx": "skus[*].name",
		"a": ["商品", "商品名称", "goods", "skus", "sku_id", "goods_id"],
		"b": ["scripts.rules.FmtGoods"]
	}`),
}

func TestColumnUnmarshalJSON(t *testing.T) {
	var normal Column
	err := jsoniter.Unmarshal(testDataColumns["normal"], &normal)
	assert.Nil(t, err)
	assert.Equal(t, "订单号", normal.Label)
	assert.Equal(t, "order_sn", normal.Name)
	assert.Equal(t, "", normal.Key)
	assert.Equal(t, []string{"订单号", "订单", "order_sn", "id"}, normal.Match)
	assert.Equal(t, []string{"scripts.rules.order_sn"}, normal.Rules)
	assert.Equal(t, false, normal.Nullable)
	assert.Equal(t, true, normal.Primary)
	assert.Equal(t, false, normal.IsArray)
	assert.Equal(t, false, normal.IsObject)

	var object Column
	err = jsoniter.Unmarshal(testDataColumns["object"], &object)
	assert.Nil(t, err)
	assert.Equal(t, "性别", object.Label)
	assert.Equal(t, "user", object.Name)
	assert.Equal(t, "sex", object.Key)
	assert.Equal(t, []string{"性别"}, object.Match)
	assert.Equal(t, []string{"scripts.rules.FmtUser"}, object.Rules)
	assert.Equal(t, true, object.Nullable)
	assert.Equal(t, false, object.IsArray)
	assert.Equal(t, true, object.IsObject)
	assert.Equal(t, false, object.Primary)

	var array Column
	err = jsoniter.Unmarshal(testDataColumns["array"], &array)
	assert.Nil(t, err)
	assert.Equal(t, "库存", array.Label)
	assert.Equal(t, "stock", array.Name)
	assert.Equal(t, "", array.Key)
	assert.Equal(t, []string{}, array.Match)
	assert.Equal(t, []string{"scripts.rules.FmtGoods"}, array.Rules)
	assert.Equal(t, false, array.Nullable)
	assert.Equal(t, true, array.IsArray)
	assert.Equal(t, false, array.IsObject)
	assert.Equal(t, false, array.Primary)

	var arrayObject Column
	err = jsoniter.Unmarshal(testDataColumns["arrayObject"], &arrayObject)
	assert.Nil(t, err)
	assert.Nil(t, err)
	assert.Equal(t, "商品", arrayObject.Label)
	assert.Equal(t, "skus", arrayObject.Name)
	assert.Equal(t, "name", arrayObject.Key)
	assert.Equal(t, []string{"商品", "商品名称", "goods", "skus", "sku_id", "goods_id"}, arrayObject.Match)
	assert.Equal(t, []string{"scripts.rules.FmtGoods"}, arrayObject.Rules)
	assert.Equal(t, false, arrayObject.Nullable)
	assert.Equal(t, true, arrayObject.IsArray)
	assert.Equal(t, true, arrayObject.IsObject)
	assert.Equal(t, false, arrayObject.Primary)

	var failure Column
	err = jsoniter.Unmarshal(testDataColumns["failure"], &failure)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), `"label" format is incorrect`)
}
