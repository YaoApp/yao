package str

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
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
