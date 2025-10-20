package seed

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

func init() {
	process.RegisterGroup("seeds", map[string]process.Handler{
		"import": processSeedImport,
	})
}

func processSeedImport(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	filename := process.ArgsString(0)
	modelName := process.ArgsString(1)

	// Default options
	options := ImportOption{
		ChunkSize: ChunkSizeDefault,
		Duplicate: DuplicateIgnore,
		Mode:      ImportModeBatch,
	}

	// Parse options if provided
	if process.NumOfArgs() > 2 {
		opts, err := getOptions(process.Args[2])
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}
		if opts.ChunkSize > 0 {
			options.ChunkSize = opts.ChunkSize
		}
		if opts.Duplicate != "" {
			options.Duplicate = opts.Duplicate
		}
		if opts.Mode != "" {
			options.Mode = opts.Mode
		}
	}

	// Import seed data
	result, err := Import(filename, modelName, options)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return result
}

// getOptions parses import options from interface
func getOptions(v interface{}) (ImportOption, error) {
	opts := ImportOption{
		ChunkSize: ChunkSizeDefault,
		Duplicate: DuplicateIgnore,
		Mode:      ImportModeBatch,
	}

	switch val := v.(type) {
	case map[string]interface{}:
		if chunkSize, exists := val["chunk_size"]; exists {
			if cs := toInt(chunkSize); cs > 0 {
				opts.ChunkSize = cs
			}
		}
		if duplicate, exists := val["duplicate"]; exists {
			if dup := toString(duplicate); dup != "" {
				opts.Duplicate = DuplicateMode(dup)
			}
		}
		if mode, exists := val["mode"]; exists {
			if m := toString(mode); m != "" {
				opts.Mode = ImportMode(m)
			}
		}

	case maps.MapStr:
		if chunkSize := val.Get("chunk_size"); chunkSize != nil {
			if cs := toInt(chunkSize); cs > 0 {
				opts.ChunkSize = cs
			}
		}
		if duplicate := val.Get("duplicate"); duplicate != nil {
			if dup := toString(duplicate); dup != "" {
				opts.Duplicate = DuplicateMode(dup)
			}
		}
		if mode := val.Get("mode"); mode != nil {
			if m := toString(mode); m != "" {
				opts.Mode = ImportMode(m)
			}
		}

	case ImportOption:
		opts = val

	default:
		return opts, nil
	}

	// Validate options
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = ChunkSizeDefault
	}

	return opts, nil
}

// toInt converts various types to int
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint8:
		return int(val)
	case uint16:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	case string:
		// Try to parse string as number
		if i, err := parseIntString(val); err == nil {
			return i
		}
	}
	return 0
}

// toString converts various types to string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	}
	return ""
}

// parseIntString parses a string to int
func parseIntString(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
