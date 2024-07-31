package runtime

import (
	"os"
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
)

func TestStart(t *testing.T) {
	testPrepare(t)
	defer Stop()
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}

func testPrepare(t *testing.T, rootEnv ...string) {

	appRootEnv := "YAO_TEST_APPLICATION"
	if len(rootEnv) > 0 {
		appRootEnv = rootEnv[0]
	}

	root := os.Getenv(appRootEnv)
	var app application.Application
	var err error

	app, err = application.OpenFromDisk(root) // Load app from Disk
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)
}
