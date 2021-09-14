package global

import (
	"os"
	"testing"

	"github.com/yaoapp/gou"
)

var cfg Config

func TestMain(m *testing.M) {

	// Run test suites
	exitVal := m.Run()

	// we can do clean up code here
	gou.KillPlugins()

	os.Exit(exitVal)
}
