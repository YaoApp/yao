//go:build unit

package plan_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/plan"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestNew(t *testing.T) {
	p := plan.New()
	assert.NotNil(t, p)
}

func TestAdd(t *testing.T) {
	p := plan.New()
	ctx := types.NewContext(nil, nil)

	err := p.Add(ctx, "member-1", map[string]interface{}{"task": "test"}, time.Now().Add(time.Hour))
	assert.NoError(t, err)
}

func TestRemove(t *testing.T) {
	p := plan.New()
	ctx := types.NewContext(nil, nil)

	err := p.Remove(ctx, "member-1", "item-123")
	assert.NoError(t, err)
}

func TestList(t *testing.T) {
	p := plan.New()
	ctx := types.NewContext(nil, nil)

	items, err := p.List(ctx, "member-1")
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestGetDue(t *testing.T) {
	p := plan.New()
	ctx := types.NewContext(nil, nil)

	items, err := p.GetDue(ctx, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}
