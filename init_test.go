package main

import (
	"os"
	"testing"
)

var cfg Config

func TestMain(m *testing.M) {
	cfg = NewConfig()

	// Run test suites
	exitVal := m.Run()

	// we can do clean up code here

	os.Exit(exitVal)
}
