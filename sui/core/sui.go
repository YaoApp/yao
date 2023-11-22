package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/maps"
)

var varRe = regexp.MustCompile(`{{\s*([^{}]+)\s*}}`)

// Setting the struct for the DSL
func (sui *DSL) Setting() (*Setting, error) {
	return &Setting{
		ID:    sui.ID,
		Guard: sui.Guard,
		Option: map[string]interface{}{
			"disableCodeEditor": false,
		},
	}, nil
}

// WithSid set the sid
func (sui *DSL) WithSid(sid string) {
	sui.Sid = sid
}

// PublicRoot returns the public root path
func (sui *DSL) PublicRoot() (string, error) {
	// Cache the public root
	if sui.publicRoot != "" {
		return sui.publicRoot, nil
	}

	ss := session.Global().ID(sui.Sid)
	data, err := ss.Dump()
	if err != nil {
		return "", err
	}

	vars := map[string]interface{}{"$session": data}
	var root = sui.Public.Root
	dot := maps.Of(vars).Dot()

	output := varRe.ReplaceAllStringFunc(root, func(matched string) string {
		varName := strings.TrimSpace(matched[2 : len(matched)-2])
		if value, ok := dot[varName]; ok {
			return fmt.Sprint(value)
		}
		return "__undefined"
	})

	sui.publicRoot = output
	return output, nil
}
