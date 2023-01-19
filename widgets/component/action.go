package component

import (
	"fmt"
	"io"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/crypto/md4"
)

// UnmarshalJSON for json UnmarshalJSON
func (action *ActionDSL) UnmarshalJSON(data []byte) error {
	var alias aliasActionDSL
	err := jsoniter.Unmarshal(data, &alias)
	if err != nil {
		return err
	}

	*action = ActionDSL(alias)
	action.ID, err = action.Hash()
	if err != nil {
		return err
	}

	//  Syntactic sugar Disabled
	if action.Disabled != nil {
		if action.Disabled.Eq != nil {
			action.Disabled.Value = action.Disabled.Eq
		}

		if action.Disabled.Equal != nil {
			action.Disabled.Value = action.Disabled.Equal
		}

		if action.Disabled.Field != "" {
			action.Disabled.Bind = fmt.Sprintf("{{%s}}", action.Disabled.Field)
		}
	}

	// Syntactic sugar { "hide": ["add", "edit", "view"] }
	// ["add", "edit", "view"]
	if action.Hide != nil {

		// set default value
		action.ShowWhenAdd = true   // shown in add form
		action.ShowWhenView = true  // shown in view form
		action.HideWhenEdit = false // shown in edit form

		for _, kind := range action.Hide {
			kind = strings.ToLower(kind)
			switch kind {
			case "add":
				action.ShowWhenAdd = false
				break
			case "view":
				action.ShowWhenView = false
				break
			case "edit":
				action.HideWhenEdit = true
				break
			}
		}
	}

	if action.Action == nil {
		action.Action = ActionNodes{}
	}

	err = action.Action.Parse()
	if err != nil {
		return err
	}

	return nil
}

// UnmarshalJSON for json UnmarshalJSON
func (nodes *ActionNodes) UnmarshalJSON(data []byte) error {

	var alias interface{}
	err := jsoniter.Unmarshal(data, &alias)
	if err != nil {
		return err
	}

	switch values := alias.(type) {

	case string: // "Actions.test.back"
		*nodes = ActionNodes{{
			"name":    values,
			"type":    values,
			"payload": map[string]interface{}{},
		}}
		return nil

	case map[string]interface{}: // {"Form.delete": {  "pathname": "/x/Table/env" }}
		node := ActionNode{}
		for name, payload := range values {
			node["name"] = name
			node["type"] = name
			node["payload"] = payload
			break
		}
		*nodes = ActionNodes{node}
		return nil

	case []interface{}, []map[string]interface{}: //  [{ "name": "Save", "type": "Form.save",  "payload": { "id": ":id", "status": "cured" }}]
		new := aliasActionNodes{}
		err := jsoniter.Unmarshal(data, &new)
		if err != nil {
			return err
		}

		*nodes = ActionNodes(new)
		return nil

		// case []ActionNode:
		// 	*nodes = ActionNodes(values)
		// 	return nil

		// case ActionNodes:
		// 	*nodes = values
		// 	return nil

		// case *ActionNodes:
		// 	nodes = values
		// 	return nil
	}

	return fmt.Errorf("the format does not support. %s", string(data))
}

// MarshalJSON for json MarshalJSON
// func (nodes ActionNodes) MarshalJSON() ([]byte, error) {
// 	return nil, nil
// }

// Parse the custom nodes
func (nodes *ActionNodes) Parse() error {
	for i := range *nodes {
		// merge the developer-defined actions
		if (*nodes)[i].Custom() {
			// (*nodes)[i]["custom"] = true
		}
	}
	return nil
}

// Custom check if the action node is custom
func (node ActionNode) Custom() bool {
	if name, ok := node["type"].(string); ok && strings.HasPrefix(strings.ToLower(name), "actions.") {
		return true
	}
	return false
}

// Hash hash value
func (action ActionDSL) Hash() (string, error) {
	h := md4.New()
	origin := fmt.Sprintf("ACTION::%#v", action.Action)
	// fmt.Println("Origin:", origin)
	io.WriteString(h, origin)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Filter actions
func (actions Actions) Filter(excludes map[string]bool) Actions {
	new := []ActionDSL{}
	for _, action := range actions {
		if _, has := excludes[action.ID]; !has {
			new = append(new, action)
		}
	}
	return new
}

// SetPath set actions xpath
// func (actions Actions) SetPath(root string) Actions {
// 	for i := range actions {
// 		actions[i].Xpath = fmt.Sprintf("%s.%d", root, i)
// 	}
// 	return actions
// }
