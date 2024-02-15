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
	err := application.Parse("<source>.yao", source, &pipe)
	if err != nil {
		return nil, fmt.Errorf("parse pipe: %s", err)
	}

	err = (&pipe).build()
	if err != nil {
		return nil, fmt.Errorf("build pipe: %s", err)
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

	if pipe.Nodes == nil || len(pipe.Nodes) == 0 {
		return fmt.Errorf("pipe: %s nodes is required", pipe.Name)
	}

	return pipe._build()
}

// HasNodes check if the pipe has nodes
func (pipe *Pipe) HasNodes() bool {
	return pipe.Nodes != nil && len(pipe.Nodes) > 0
}

func (pipe *Pipe) _build() error {

	pipe.mapping = map[string]*Node{}
	if pipe.Nodes == nil {
		return nil
	}

	for i, node := range pipe.Nodes {
		if node.Name == "" {
			return fmt.Errorf("pipe: %s nodes[%d] name is required", pipe.Name, i)
		}

		pipe.Nodes[i].index = i
		pipe.mapping[node.Name] = &pipe.Nodes[i]

		// Set the label of the node
		if node.Label == "" {
			pipe.Nodes[i].Label = strings.ToUpper(node.Name)
		}

		// Set the type of the node
		if node.Process != nil {
			pipe.Nodes[i].Type = "process"

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
			continue

		} else if node.Request != nil {
			pipe.Nodes[i].Type = "request"
			continue

		} else if node.Prompts != nil {
			pipe.Nodes[i].Type = "ai"
			continue

		} else if node.UI != "" {
			pipe.Nodes[i].Type = "user-input"
			if node.UI != "cli" && node.UI != "web" && node.UI != "app" && node.UI != "wxapp" { // Vaildate the UI type
				return fmt.Errorf("pipe: %s nodes[%d] the type of the UI must be cli, web, app, wxapp", pipe.Name, i)
			}
			continue

		} else if node.Switch != nil {
			pipe.Nodes[i].Type = "switch"
			for key, pip := range node.Switch {
				key = ref(key)
				pip.Whitelist = pipe.Whitelist // Copy the whitelist
				pip.namespace = node.Name
				pip.parent = pipe
				if pip.ID == "" {
					pip.ID = fmt.Sprintf("%s.%s#%s", pipe.ID, node.Name, key)
				}
				if pip.Name == "" {
					pip.Name = fmt.Sprintf("%s(%s#%s)", pipe.Name, node.Name, key)
				}
				pip._build()
			}
			continue
		}

		return fmt.Errorf("pipe: %s nodes[%d] process, request, case, prompts or ui is required at least one", pipe.Name, i)
	}

	return nil
}
