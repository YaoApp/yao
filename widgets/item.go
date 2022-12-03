package widgets

import (
	"os"
	"sort"
	"strings"
)

// Item the item
type Item struct {
	Name     string      `json:"name,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Children []Item      `json:"children,omitempty"`
}

// Sort items
func Sort(items []Item, orders []string) {

	rank := map[string]int{}
	if orders != nil {
		for i, name := range orders {
			rank[name] = i
		}
	}

	sort.Slice(items, func(i, j int) bool {
		rankI, hasI := rank[items[i].Name]
		rankJ, hasJ := rank[items[j].Name]
		if hasI && hasJ {
			return rankI < rankJ
		}
		return strings.Compare(items[i].Name, items[j].Name) < 0
	})

	// Sort Children
	for i := range items {
		if len(items[i].Children) > 0 {
			Sort(items[i].Children, nil)
		}
	}

}

// Grouping by name
func Grouping(items map[string]interface{}) map[string]interface{} {
	grouping := map[string]interface{}{}
	for name, item := range items {
		paths := strings.Split(name, string(os.PathSeparator))
		node := grouping
		for _, path := range paths {
			if strings.HasSuffix(path, ".json") {
				node[path] = Item{Name: path, Data: item, Children: []Item{}}
				continue
			}
			if _, has := node[path]; !has {
				node[path] = map[string]interface{}{"name": path, "data": map[string]interface{}{}}
			}
			node = node[path].(map[string]interface{})
		}
	}
	return grouping
}

// Array to Array
func Array(groupingItems map[string]interface{}, res []Item) []Item {

	for key, item := range groupingItems {

		switch it := item.(type) {

		case map[string]interface{}: // Path
			if it["name"] == nil {
				break
			}
			res = append(res, Item{
				Name:     key,
				Data:     nil,
				Children: Array(it, []Item{}),
			})
			break

		case Item: // Data
			res = append(res, it)
			break
		}
	}

	return res
}
