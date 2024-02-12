package pipe

import (
	"errors"
	"fmt"

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
