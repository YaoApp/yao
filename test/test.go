package test

import (
	"os"
	"testing"

	"github.com/yaoapp/gou/application"
)

// Prepare test environment
func Prepare(t *testing.T) {
	root := os.Getenv("YAO_TEST_APPLICATION")
	var app application.Application
	var err error

	if root == "bin:application.pkg" {
		key := os.Getenv("YAO_TEST_PRIVATE_KEY")
		app, err = application.OpenFromBin(root, key) // Load app from Bin
		if err != nil {
			t.Fatal(err)
		}
		application.Load(app)
		return
	}

	app, err = application.OpenFromDisk(root) // Load app from Disk
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)
}

// Clean the test environment
func Clean() {}
