package component

import (
	"fmt"
	"io"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/crypto/md4"
)

// SetPath set actions xpath
func (actions Actions) SetPath(root string) Actions {
	for i := range actions {
		actions[i].Xpath = fmt.Sprintf("%s.%d", root, i)
	}
	return actions
}

// Hash hash value
func (actions Actions) Hash() Actions {
	h := md4.New()
	for i := range actions {
		keys := []string{}
		for key, value := range actions[i].Action {
			for k, v := range value {
				data, _ := jsoniter.Marshal(v)
				key = fmt.Sprintf("%s%s%s", key, k, data)
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		origin := strings.Join(keys, "|")
		io.WriteString(h, origin)
		actions[i].ID = fmt.Sprintf("%x", h.Sum(nil))
	}
	return actions
}
