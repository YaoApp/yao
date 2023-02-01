package main

import (
	"os"
	"testing"

	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/yao/config"
)

var cfg config.Config

func TestMain(m *testing.M) {

	// Run test suites
	exitVal := m.Run()

	// we can do clean up code here
	plugin.KillAll()

	os.Exit(exitVal)
}
