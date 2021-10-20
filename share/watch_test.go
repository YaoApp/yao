package share

import (
	"log"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/config"
)

func TestWatch(t *testing.T) {
	root := path.Join(config.Conf.Source, "/tests/flows")
	assert.NotPanics(t, func() {
		go Watch(root, func(op string, file string) {
			log.Println(op, file)
		})
		time.Sleep(time.Second * 2)
		defer StopWatch()
	})
}
