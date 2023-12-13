package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/sui/storages/azure"
	"github.com/yaoapp/yao/sui/storages/local"
)

// New create a new sui
func New(dsl *core.DSL) (core.SUI, error) {

	if dsl.Storage == nil {
		return nil, fmt.Errorf("storage is not required")
	}

	switch strings.ToLower(dsl.Storage.Driver) {

	case "local":
		return local.New(dsl)

	case "azure":
		return azure.New(dsl)

	default:
		return nil, fmt.Errorf("%s is not a valid driver", dsl.Storage.Driver)
	}
}

// Load load the sui
func Load(cfg config.Config) error {
	exts := []string{"*.sui.yao", "*.sui.jsonc", "*.sui.json"}
	err := application.App.Walk("suis", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := share.ID(root, file)
		_, err := loadFile(file, id)
		if err != nil {
			log.Error("[sui] Load sui %s error: %s", id, err.Error())
			return nil
		}
		return nil
	}, exts...)

	if err != nil {
		return err
	}

	buildRouteMatchers()
	return registerAPI()
}

func loadFile(file string, id string) (core.SUI, error) {

	dsl, err := core.Load(file, id)
	if err != nil {
		return nil, err
	}

	sui, err := New(dsl)
	if err != nil {
		return nil, err
	}

	core.SUIs[id] = sui
	return core.SUIs[id], nil
}

// Reload reload the route matchers
func Reload() {
	buildRouteMatchers()
}

func buildRouteMatchers() (map[*regexp.Regexp][][]*core.Matcher, map[string][][]*core.Matcher) {
	matchers := map[*regexp.Regexp][][]*core.Matcher{}
	exactMatchers := map[string][][]*core.Matcher{}
	for id, sui := range core.SUIs {
		suiMatcher := sui.PublicRootMatcher()
		if suiMatcher.Regex != nil {
			matchers[suiMatcher.Regex] = [][]*core.Matcher{}

		} else if suiMatcher.Exact != "" {
			exactMatchers[suiMatcher.Exact] = [][]*core.Matcher{}

		} else {
			log.Error("[sui] Load sui %s error: %s", id, "the public root is empty")
			continue
		}

		tmpls, err := sui.GetTemplates()
		if err != nil {
			log.Error("[sui] Load sui %s error: %s", id, err.Error())
			continue
		}

		for _, tmpl := range tmpls {
			pages, err := tmpl.Pages()
			if err != nil {
				log.Error("[sui] Load sui %s error: %s", id, err.Error())
				continue
			}

			for _, page := range pages {
				route := page.Get().Route
				parts := strings.Split(route, "/")[1:]

				for i, part := range parts {
					parent := ""
					if i > 0 {
						parent = parts[i-1]
					}
					matcher := &core.Matcher{Ref: part, Parent: parent}
					if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
						matcher.Regex = core.RouteRegexp
					} else {
						matcher.Exact = part
					}

					if suiMatcher.Regex != nil {
						if len(matchers[suiMatcher.Regex]) < i+1 {
							matchers[suiMatcher.Regex] = append(matchers[suiMatcher.Regex], []*core.Matcher{})
						}
						matchers[suiMatcher.Regex][i] = append(matchers[suiMatcher.Regex][i], matcher)
					}

					if suiMatcher.Exact != "" {
						if len(exactMatchers[suiMatcher.Exact]) < i+1 {
							exactMatchers[suiMatcher.Exact] = append(exactMatchers[suiMatcher.Exact], []*core.Matcher{})
						}
						exactMatchers[suiMatcher.Exact][i] = append(exactMatchers[suiMatcher.Exact][i], matcher)
					}
				}
			}
		}
	}

	core.RouteMatchers = matchers
	core.RouteExactMatchers = exactMatchers
	return matchers, exactMatchers
}
