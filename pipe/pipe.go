package pipe

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

var pipes = map[string]*Pipe{}

// Load the pipe
func Load(cfg config.Config) error {

	exts := []string{"*.pip.yao", "*.pipe.yao"}
	errs := []error{}
	err := application.App.Walk("pipes", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := share.ID(root, file)
		pipe, err := NewFile(file, root)
		if err != nil {
			errs = append(errs, err)
			return err
		}

		Set(id, pipe)
		return err
	}, exts...)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return err
}

// New create Pipe
func New(source []byte) (*Pipe, error) {
	pipe := Pipe{}
	err := application.Parse("<source>", source, &pipe)
	if err != nil {
		return nil, err
	}

	err = (&pipe).build()
	if err != nil {
		return nil, err
	}

	return &pipe, nil
}

// NewFile create pipe from file
func NewFile(file string, root string) (*Pipe, error) {
	source, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	id := share.ID(root, file)
	pipe := Pipe{ID: id}
	err = application.Parse(file, source, &pipe)
	if err != nil {
		return nil, err
	}

	err = (&pipe).build()
	if err != nil {
		return nil, err
	}

	return &pipe, nil
}

// Set pipe to
func Set(id string, pipe *Pipe) {
	pipes[id] = pipe
}

// Remove the pipe
func Remove(id string) {
	if _, has := pipes[id]; has {
		delete(pipes, id)
	}
}

// Get the pipe
func Get(id string) (*Pipe, error) {
	if pipe, has := pipes[id]; has {
		return pipe, nil
	}
	return nil, fmt.Errorf("pipe %s not found", id)
}

// Build the pipe
func (pipe *Pipe) build() error {
	pipe.mapping = map[string]*Node{}
	if pipe.Nodes == nil || len(pipe.Nodes) == 0 {
		return fmt.Errorf("pipe: %s nodes is required", pipe.Name)
	}

	return pipe._build("", pipe.Nodes)
}

func (pipe *Pipe) _build(namespace string, nodes []Node) error {

	for i, node := range nodes {
		if node.Name == "" {
			return fmt.Errorf("pipe: %s nodes[%d] name is required", pipe.Name, i)
		}

		name := node.Name
		if namespace != "" {
			name = namespace + "." + name
		}

		// Set the index of the node
		if nodes[i].index == nil {
			nodes[i].index = []int{}
		}

		nodes[i].index = append(nodes[i].index, i)
		nodes[i].namespace = namespace
		pipe.mapping[name] = &nodes[i]

		// Set the label of the node
		if node.Label == "" {
			nodes[i].Label = strings.ToUpper(node.Name)
		}

		// Set the type of the node
		if node.Process != nil {
			nodes[i].Type = "process"

			// Validate the process
			if node.Process.Name == "" {
				return fmt.Errorf("pipe: %s nodes[%d] process name is required", pipe.Name, i)
			}

			// Security check
			if pipe.Whitelist != nil {
				if _, has := pipe.Whitelist[node.Process.Name]; !has {
					return fmt.Errorf("pipe: %s nodes[%d] process %s is not in the whitelist", pipe.Name, i, node.Process.Name)
				}
			}

		} else if node.Request != nil {
			nodes[i].Type = "request"

		} else if node.Prompts != nil {
			nodes[i].Type = "ai"

		} else if node.Case != nil {
			nodes[i].Type = "switch"
			for _, sub := range node.Case {
				// Copy the whitelist to the sub pipe
				sub.Name = fmt.Sprintf("%s.%s", pipe.Name, node.Name)
				sub.Whitelist = pipe.Whitelist
				sub.mapping = map[string]*Node{}
				sub.namespace = node.Name
				if sub.Nodes != nil && len(sub.Nodes) > 0 {
					err := sub._build("", sub.Nodes)
					if err != nil {
						return err
					}
				}
			}

		} else if node.UI != "" {
			nodes[i].Type = "user-input"
			if node.UI != "cli" && node.UI != "web" && node.UI != "app" && node.UI != "wxapp" { // Vaildate the UI type
				return fmt.Errorf("pipe: %s nodes[%d] the type of the UI must be cli, web, app, wxapp", pipe.Name, i)
			}

		} else {
			return fmt.Errorf("pipe: %s nodes[%d] process, request, case, prompts or ui is required at least one", pipe.Name, i)
		}
	}

	return nil
}
