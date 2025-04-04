package str

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mozillazg/go-pinyin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

// ProcessJoin utils.str.Join
func ProcessJoin(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	args := process.ArgsArray(0)
	sep := process.ArgsString(1)
	strs := []string{}
	for i := range args {
		strs = append(strs, fmt.Sprintf("%v", args[i]))
	}
	return strings.Join(strs, sep)
}

// ProcessJoinPath utils.str.JoinPath
func ProcessJoinPath(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	paths := []string{}
	for _, arg := range process.Args {
		paths = append(paths, fmt.Sprintf("%v", arg))
	}
	return filepath.Join(paths...)
}

// ProcessUUID utils.str.uuid
func ProcessUUID(process *process.Process) interface{} {
	uuid := uuid.New()
	return uuid.String()
}

// ProcessPinyin utils.str.Pinyin converts Chinese characters to Pinyin
// Args:
//   - arg[0]: string, the Chinese characters to convert
//   - arg[1]: map (optional) configuration options
//     {
//     "tone": bool,     // whether to include tone marks, default: false
//     "separator": string  // separator between pinyin, default: " "
//     }
func ProcessPinyin(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	seed := process.ArgsString(0)

	// Create converter with settings
	a := pinyin.NewArgs()
	a.Style = pinyin.Normal // default style
	separator := " "        // default separator

	// Apply custom settings if provided
	if process.NumOfArgs() > 1 {
		confMap := process.ArgsMap(1, maps.MapStrAny{})

		// Check if tone is enabled
		toneVal, hasTone := confMap["tone"]

		if hasTone {
			if tone, ok := toneVal.(bool); ok && tone {
				a.Style = pinyin.Tone
			}
		}

		// Check custom separator
		if sep, ok := confMap["separator"].(string); ok {
			separator = sep
		}
	}

	// Convert to Pinyin
	result := pinyin.Pinyin(seed, a)

	// Convert 2D array to 1D array
	pinyinStr := make([]string, 0, len(result))
	for _, py := range result {
		if len(py) > 0 {
			pinyinStr = append(pinyinStr, py[0])
		}
	}

	final := strings.Join(pinyinStr, separator)
	return final
}
