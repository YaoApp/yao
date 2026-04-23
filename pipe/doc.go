package pipe

import (
	_ "embed"
	"github.com/yaoapp/gou/doc"
)

//go:embed doc.yml
var docYAML []byte

//go:embed doc_pipes.yml
var docPipesYAML []byte

func init() {
	doc.LoadYAML(docYAML)
	doc.LoadYAML(docPipesYAML)
}
