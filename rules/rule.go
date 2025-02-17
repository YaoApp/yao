package rules

import "github.com/yaoapp/yao/helper"

// rulesToMaps converts a slice of Rule to a slice of maps
func rulesToMaps(rules []Rule, onlyMenu bool, keys []string) []map[string]interface{} {
	maps := []map[string]interface{}{}
	for _, rule := range rules {
		if onlyMenu && rule.Path == "" && len(rule.Children) == 0 {
			continue
		}
		if !helper.ContainsString(keys, rule.Key) && !helper.ContainsString(keys, "*") {
			continue
		}
		maps = append(maps, rule.ToMap(onlyMenu, keys))
	}
	return maps
}

// ToMap converts a Rule to a map
func (rule *Rule) ToMap(onlyMenu bool, keys []string) map[string]interface{} {
	return map[string]interface{}{
		"id":           rule.ID,
		"name":         rule.Name,
		"title":        rule.Name,
		"icon":         rule.Icon,
		"path":         rule.Path,
		"visible_menu": rule.Visible_menu,
		"children":     rulesToMaps(rule.Children, onlyMenu, keys),
		"rule":         rule.Key,
	}
}

func (dsl *DSL) HasChildren(onlyMenu bool, keys []string) bool {
	ruls := []Rule{}
	for _, rule := range dsl.Children {
		if onlyMenu && rule.Path == "" && len(rule.Children) == 0 {
			continue
		}
		if !helper.ContainsString(keys, rule.Key) && !helper.ContainsString(keys, "*") {
			continue
		}
		ruls = append(ruls, rule)
	}
	return len(ruls) > 0
}

// GetDSLsMaps returns the maps of DSLs corresponding to the given IDs
func GetDSLsMaps(ids []string, onlyMenu bool, keys []string) []map[string]interface{} {
	if keys == nil {
		keys = []string{"*"}
	}
	maps := []map[string]interface{}{}
	for _, id := range ids {
		if dsl, ok := RuleDSLS[id]; ok {
			rule := dsl.ToMap(onlyMenu, keys)
			if dsl.HasChildren(onlyMenu, keys) {
				maps = append(maps, rule)
			}
		}
	}
	return maps
}
