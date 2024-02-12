package pipe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestRun(t *testing.T) {
	prepare(t)
	defer test.Clean()
	translator, err := Get("translator")
	if err != nil {
		t.Fatal(err)
	}

	sid := session.ID()
	context, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx := translator.
		Create().
		With(context).
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid(sid)
	defer Close(ctx.ID())

	assert.NotPanics(t, func() { ctx.Run() })
}

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
