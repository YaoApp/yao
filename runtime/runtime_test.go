package runtime

import (
	"testing"

	"github.com/yaoapp/yao/config"
)

func TestStart(t *testing.T) {
	defer Stop()
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
