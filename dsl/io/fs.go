package io

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/dsl/types"
)

// FS is the fs io
type FS struct {
	Type types.Type
}

// NewFS create a new fs io
func NewFS(typ types.Type) types.IO {
	return &FS{Type: typ}
}

// Inspect get the info from the file
func (fs *FS) Inspect(id string) (*types.Info, bool, error) {
	file := types.ToPath(fs.Type, id)
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

	// Parse the source to extract metadata
	var sourceData map[string]interface{}
	err = application.Parse(file, data, &sourceData)
	if err != nil {
		return nil, true, err
	}

	// Extract common fields from source
	var label, description string
	var tags []string
	var sort int

	if v, ok := sourceData["label"]; ok {
		if s, ok := v.(string); ok {
			label = s
		}
	}

	if v, ok := sourceData["description"]; ok {
		if s, ok := v.(string); ok {
			description = s
		}
	}

	if v, ok := sourceData["tags"]; ok {
		if tagsList, ok := v.([]interface{}); ok {
			for _, tag := range tagsList {
				if s, ok := tag.(string); ok {
					tags = append(tags, s)
				}
			}
		}
	}

	if v, ok := sourceData["sort"]; ok {
		if s, ok := v.(float64); ok {
			sort = int(s)
		}
	}

	// Get file info for timestamps
	fileInfo, err := application.App.Info(file)
	if err != nil {
		return nil, true, err
	}

	// Create Info structure with correct fields
	info := &types.Info{
		ID:          id,
		Type:        fs.Type,
		Label:       label,
		Description: description,
		Tags:        tags,
		Sort:        sort,
		Path:        file,
		Store:       types.StoreTypeFile,
		Readonly:    false,
		Builtin:     false,
		Status:      types.StatusLoading,
		Mtime:       fileInfo.ModTime(),
		Ctime:       fileInfo.ModTime(),
	}

	return info, true, nil
}

// Source get the source from the file
func (fs *FS) Source(id string) (string, bool, error) {
	path := types.ToPath(fs.Type, id)
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

// List get the list from the path
func (fs *FS) List(options *types.ListOptions) ([]*types.Info, error) {
	root, exts := types.TypeRootAndExts(fs.Type)
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
		id := types.WithTypeToID(fs.Type, file)
		info, _, err := fs.Inspect(id)
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
						if options.Source {
							source, _, err := fs.Source(id)
							if err != nil {
								errs = append(errs, err)
								return nil
							}
							info.Source = source
						}
						infos = append(infos, info)
						return nil
					}
				}
			}
		}

		// Add to the list
		if options.Source {
			source, _, err := fs.Source(id)
			if err != nil {
				errs = append(errs, err)
				return nil
			}
			info.Source = source
		}
		infos = append(infos, info)
		return err
	}, patterns...)

	return infos, err
}

// Create create the file
func (fs *FS) Create(options *types.CreateOptions) error {

	path := types.ToPath(fs.Type, options.ID)

	// Check if the file is a directory
	exists, err := application.App.Exists(path)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("%v %s already exists", fs.Type, options.ID)
	}

	// Create the file
	return application.App.Write(path, []byte(options.Source))
}

// Update update the file
func (fs *FS) Update(options *types.UpdateOptions) error {

	// Validate the options
	if options.Source == "" && options.Info == nil {
		return fmt.Errorf("%v %s one of source or info is required", fs.Type, options.ID)
	}

	path := types.ToPath(fs.Type, options.ID)

	// Check if the file exists
	exists, err := application.App.Exists(path)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%v %s not found", fs.Type, options.ID)
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

// Delete delete the file
func (fs *FS) Delete(id string) error {

	path := types.ToPath(fs.Type, id)

	// Check if the file is a directory
	exists, err := application.App.Exists(path)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%v %s not found", fs.Type, id)
	}

	// Delete the file
	return application.App.Remove(path)
}

// Exists check if the file exists
func (fs *FS) Exists(id string) (bool, error) {
	path := types.ToPath(fs.Type, id)
	return application.App.Exists(path)
}
