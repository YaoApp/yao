package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
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

// GetSid returns the sid
func (sui *DSL) GetSid() string {
	return sui.Sid
}

// PublicRootMatcher returns the public root matcher
func (sui *DSL) PublicRootMatcher() *Matcher {
	pub := sui.GetPublic()
	if varRe.MatchString(pub.Root) {
		if pub.Matcher != "" {
			re, err := regexp.Compile(pub.Matcher)
			if err != nil {
				log.Error("[sui] %s matcher error %s, use the default matcher", sui.ID, err.Error())
				return &Matcher{Regex: RouteRegexp}
			}
			return &Matcher{Regex: re}
		}
		return &Matcher{Regex: RouteRegexp}
	}
	return &Matcher{Exact: pub.Root}
}

// PublicRootWithSid returns the public root path with sid
func (sui *DSL) PublicRootWithSid(sid string) (string, error) {
	ss := session.Global().ID(sid)
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

	return output, nil
}

// PublicRoot returns the public root path
func (sui *DSL) PublicRoot(data map[string]interface{}) (string, error) {
	// Cache the public root (Close the cache)
	// if sui.publicRoot != "" {
	// 	return sui.publicRoot, nil
	// }

	if data == nil {
		data = map[string]interface{}{}
	}

	ss := session.Global().ID(sui.Sid)
	sessionData, err := ss.Dump()
	if err != nil {
		return "", err
	}

	// Merge the session data
	if sessionData == nil {
		sessionData = map[string]interface{}{}
	}

	// Merge the data
	for k, v := range sessionData {
		if _, ok := data[k]; !ok {
			data[k] = v
		}
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

// GetTemplate returns the template
func (sui *DSL) GetTemplate(name string) (ITemplate, error) {
	return nil, nil
}

// GetTemplates returns the templates
func (sui *DSL) GetTemplates() ([]ITemplate, error) {
	return nil, nil
}

// UploadTemplate upload the template
func (sui *DSL) UploadTemplate(src string, dst string) (ITemplate, error) {
	return nil, nil
}

// GetPublic returns the public
func (sui *DSL) GetPublic() *Public {
	return sui.Public
}
