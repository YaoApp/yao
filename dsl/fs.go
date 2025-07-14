package dsl

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/dsl/types"
)

// getInfoFromFile get the info from the file
func (dsl *DSL) fsInspect(id string, path ...string) (*types.Info, bool, error) {
	file := ""
	if len(path) > 0 {
		file = path[0]
	} else {
		file = types.ToPath(dsl.Type, id)
	}

	var info types.Info = types.Info{ID: id, Path: file}
	exists, err := application.App.Exists(file)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	// Read the file
	data, err := application.App.Read(file)
	if err != nil {
		return nil, false, err
	}

	// Unmarshal the data to the info
	err = application.Parse(file, data, &info)
	if err != nil {
		return nil, true, err
	}

	// Merge the mtime and ctime
	fileInfo, err := application.App.Info(file)
	if err != nil {
		return nil, true, err
	}

	info.Mtime = fileInfo.ModTime()
	info.Ctime = fileInfo.ModTime()
	return &info, true, nil
}

// getSourceFromFile get the source from the file
func (dsl *DSL) fsSource(id string) (string, bool, error) {
	path := types.ToPath(dsl.Type, id)
	exists, err := application.App.Exists(path)
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, nil
	}

	// Read the file
	data, err := application.App.Read(path)
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}

// getListFromPath get the list from the path
func (dsl *DSL) fsList(options *types.ListOptions) ([]*types.Info, error) {
	root, exts := types.TypeRootAndExts(dsl.Type)
	var infos []*types.Info = []*types.Info{}
	patterns := []string{}
	for _, ext := range exts {
		patterns = append(patterns, "*"+ext)
	}
	var errs []error
	err := application.App.Walk(root, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		id := types.WithTypeToID(dsl.Type, file)
		info, _, err := dsl.fsInspect(id, file)
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		// Filter by options
		if len(options.Tags) > 0 {
			if len(info.Tags) == 0 {
				return nil
			}

			for _, tag := range options.Tags {
				for _, t := range info.Tags {
					if t == tag {
						infos = append(infos, info)
						return nil
					}
				}
			}
		}

		// Add to the list
		infos = append(infos, info)
		return err
	}, patterns...)

	return infos, err
}

func (dsl *DSL) fsCreate(options *types.CreateOptions) error {

	path := types.ToPath(dsl.Type, options.ID)

	// Check if the file is a directory
	exists, err := application.App.Exists(path)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("%v %s already exists", dsl.Type, options.ID)
	}

	// Create the file
	return application.App.Write(path, []byte(options.Source))
}

func (dsl *DSL) fsUpdate(options *types.UpdateOptions) error {

	// Validate the options
	if options.Source == "" && options.Info == nil {
		return fmt.Errorf("%v %s one of source or info is required", dsl.Type, options.ID)
	}

	path := types.ToPath(dsl.Type, options.ID)

	// Check if the file exists
	exists, err := application.App.Exists(path)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%v %s not found", dsl.Type, options.ID)
	}

	// Update source
	if options.Source != "" {
		return application.App.Write(path, []byte(options.Source))
	}

	// Update info
	var source map[string]interface{}
	data, err := application.App.Read(path)
	if err != nil {
		return err
	}

	err = application.Parse(path, data, &source)
	if err != nil {
		return err
	}

	// Update the info
	source["id"] = options.ID
	source["label"] = options.Info.Label
	source["tags"] = options.Info.Tags
	source["description"] = options.Info.Description
	new, err := jsoniter.MarshalIndent(source, "", "  ")
	if err != nil {
		return err
	}

	return application.App.Write(path, []byte(new))
}

func (dsl *DSL) fsDelete(id string) error {

	path := types.ToPath(dsl.Type, id)

	// Check if the file is a directory
	exists, err := application.App.Exists(path)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%v %s not found", dsl.Type, id)
	}

	// Delete the file
	return application.App.Remove(path)
}

func (dsl *DSL) fsExists(id string) (bool, error) {
	path := types.ToPath(dsl.Type, id)
	return application.App.Exists(path)
}
