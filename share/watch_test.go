package share

import (
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWatch(t *testing.T) {
	root := path.Join(os.Getenv("YAO_DEV"), "/tests/flows")
	assert.NotPanics(t, func() {
		go Watch(root, func(op string, file string) {
			log.Println(op, file)
		})
		time.Sleep(time.Second * 2)
		defer StopWatch()
	})
}
