package setup

import (
	"os"
	"testing"
)

func TestInstall(t *testing.T) {
	// err := Install()
	// if err != nil {
	// 	t.Fatal(err)
	// }
}

func TestMakeInit(t *testing.T) {
	root := prepare(t)
	err := makeInit(root)
	if err != nil {
		t.Fatal(err)
	}
}

func prepare(t *testing.T) string {
	dir, err := os.MkdirTemp("", "-install")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
