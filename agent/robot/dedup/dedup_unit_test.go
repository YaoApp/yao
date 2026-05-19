//go:build unit

package dedup_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/dedup"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestNew(t *testing.T) {
	d := dedup.New()
	assert.NotNil(t, d)
}

func TestCheck(t *testing.T) {
	d := dedup.New()
	ctx := types.NewContext(nil, nil)

	t.Run("always_proceed", func(t *testing.T) {
		result, err := d.Check(ctx, "member-1", types.TriggerClock)
		assert.NoError(t, err)
		assert.Equal(t, types.DedupProceed, result)
	})

	t.Run("different_triggers", func(t *testing.T) {
		result, err := d.Check(ctx, "member-1", types.TriggerHuman)
		assert.NoError(t, err)
		assert.Equal(t, types.DedupProceed, result)

		result, err = d.Check(ctx, "member-1", types.TriggerEvent)
		assert.NoError(t, err)
		assert.Equal(t, types.DedupProceed, result)
	})

	t.Run("different_members", func(t *testing.T) {
		result, err := d.Check(ctx, "member-2", types.TriggerClock)
		assert.NoError(t, err)
		assert.Equal(t, types.DedupProceed, result)
	})
}

func TestMark(t *testing.T) {
	d := dedup.New()
	assert.NotPanics(t, func() {
		d.Mark("member-1", types.TriggerClock, 5*time.Minute)
	})
}
