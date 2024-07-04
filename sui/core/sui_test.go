package core

import (
	"path/filepath"
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func testDataPage(t *testing.T) *Page {
	root := "/data/test-cases/basic"
	path := filepath.Join(root, "data")

	document, err := application.App.Read(root + "/__document.html")
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	html, err := application.App.Read(path + "/data.html")
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	data, err := application.App.Read(path + "/data.json")
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	return &Page{
		Name:     "data",
		Route:    "data",
		Document: document,
		Codes: SourceCodes{
			HTML: Source{File: "data.html", Code: string(html)},
			DATA: Source{File: "data.json", Code: string(data)},
		},
	}
}

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf, "YAO_SUI_TEST_APPLICATION")
}

func clean() {
	test.Clean()
}
