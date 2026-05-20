//go:build unit

package search_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search"
	"github.com/yaoapp/yao/agent/search/handlers/db"
	"github.com/yaoapp/yao/agent/search/handlers/kb"
	"github.com/yaoapp/yao/agent/search/handlers/web"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestNewRegistry(t *testing.T) {
	r := search.NewRegistry()
	assert.NotNil(t, r)

	h, ok := r.Get(types.SearchTypeWeb)
	assert.False(t, ok)
	assert.Nil(t, h)
}

func TestRegistry_Register(t *testing.T) {
	r := search.NewRegistry()

	webHandler := web.NewHandler("builtin", nil)
	r.Register(webHandler)

	h, ok := r.Get(types.SearchTypeWeb)
	assert.True(t, ok)
	assert.Equal(t, types.SearchTypeWeb, h.Type())
}

func TestRegistry_RegisterMultiple(t *testing.T) {
	r := search.NewRegistry()

	r.Register(web.NewHandler("builtin", nil))
	r.Register(kb.NewHandler(nil))
	r.Register(db.NewHandler("builtin", nil))

	webH, ok := r.Get(types.SearchTypeWeb)
	assert.True(t, ok)
	assert.Equal(t, types.SearchTypeWeb, webH.Type())

	kbH, ok := r.Get(types.SearchTypeKB)
	assert.True(t, ok)
	assert.Equal(t, types.SearchTypeKB, kbH.Type())

	dbH, ok := r.Get(types.SearchTypeDB)
	assert.True(t, ok)
	assert.Equal(t, types.SearchTypeDB, dbH.Type())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := search.NewRegistry()

	h, ok := r.Get(types.SearchTypeWeb)
	assert.False(t, ok)
	assert.Nil(t, h)
}

func TestRegistry_RegisterOverwrite(t *testing.T) {
	r := search.NewRegistry()

	h1 := web.NewHandler("builtin", nil)
	r.Register(h1)

	h2 := web.NewHandler("agent", nil)
	r.Register(h2)

	h, ok := r.Get(types.SearchTypeWeb)
	assert.True(t, ok)
	assert.NotNil(t, h)
}
