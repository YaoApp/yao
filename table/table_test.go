package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {

	table, has := Tables["service"]
	assert.Equal(t, table.Table, "service")
	assert.True(t, has)

	_, has = table.Columns["id"]
	assert.True(t, has)

	price, has := table.Columns["计费方式"]
	assert.True(t, has)
	if has {
		assert.True(t, price.Edit.Props["multiple"].(bool))
	}

	_, has = table.Filters["id"]
	assert.True(t, has)

	keywords, has := table.Filters["关键词"]
	assert.True(t, has)
	if has {
		assert.True(t, keywords.Bind == "where.name.like")
	}
}
