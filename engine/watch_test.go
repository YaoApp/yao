package engine

import (
	"log"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
)

func TestWatch(t *testing.T) {
	root := path.Join(config.Conf.Source, "/app/flows")
	assert.NotPanics(t, func() {
		go share.Watch(root, func(op string, file string) {
			log.Println(op, file)
		})
		time.Sleep(time.Second * 2)
		defer share.StopWatch()
	})
}
