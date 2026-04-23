package user

import (
	_ "embed"

	"github.com/yaoapp/gou/doc"
)

//go:embed doc.yml
var docYAML []byte

func init() { doc.LoadYAML(docYAML) }
