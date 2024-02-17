package pipe

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("pipes", processPipes)
	process.RegisterGroup("pipe", map[string]process.Handler{
		"run":        processRun,
		"create":     processCreate,
		"createwith": processCreateWith, // create with global data
		"resume":     processResume,
		"resumewith": processResumeWith, // resume with global data
		"close":      processClose,
	})
}

// processScripts
func processPipes(process *process.Process) interface{} {

	pipe, err := Get(process.ID)
	if err != nil {
		exception.New("pipes.%s not loaded", 404, process.ID).Throw()
		return nil
	}
	ctx := pipe.Create().WithGlobal(process.Global).WithSid(process.Sid)
	return ctx.Run(process.Args...)
}

// processCreate process the create pipe.create <pipe.id> [...args]
func processCreate(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	dsl := process.ArgsString(0)
	args := []any{}
	if len(process.Args) > 1 {
		args = process.Args[1:]
	}

	pipe, err := New([]byte(dsl))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	ctx := pipe.Create().WithGlobal(process.Global).WithSid(process.Sid)
	return ctx.Run(args...)
}

// processCreateWith process the create pipe.createWith <pipe.id> <global>, [...args]
func processCreateWith(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	dsl := process.ArgsString(0)
	data := process.ArgsMap(1, map[string]any{})
	args := []any{}
	if len(process.Args) > 2 {
		args = process.Args[2:]
	}

	// merge the global data
	if process.Global != nil {
		merge := map[string]any{}
		global := process.Global
		for k, v := range global {
			merge[k] = v
		}

		if data != nil {
			for k, v := range data {
				merge[k] = v
			}
		}
		data = merge
	}

	pipe, err := New([]byte(dsl))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	ctx := pipe.Create().WithGlobal(data).WithSid(process.Sid)
	return ctx.Run(args...)
}

// processRun process the resume pipe.run <pipe.id> [...args]
func processRun(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	pid := process.ArgsString(0)
	args := []any{}
	if len(process.Args) > 1 {
		args = process.Args[1:]
	}
	pipe, err := Get(pid)
	if err != nil {
		exception.New("pipes.%s not loaded", 404, process.ID).Throw()
	}

	ctx := pipe.Create().WithGlobal(process.Global).WithSid(process.Sid)
	return ctx.Run(args...)
}

// processResume process the resume pipe.resume <id> [...args]
func processResume(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	args := []any{}
	if len(process.Args) > 1 {
		args = process.Args[1:]
	}

	ctx, err := Open(id)
	if err != nil {
		exception.New("pipes.%s not found", 404, id).Throw()
	}

	return ctx.
		WithGlobal(process.Global).
		WithSid(process.Sid).
		Resume(id, args...)
}

// processResumeWith process the resume pipe.resumeWith <id> <global>, [...args]
func processResumeWith(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	id := process.ArgsString(0)
	data := process.ArgsMap(1, map[string]any{})

	args := []any{}
	if len(process.Args) > 2 {
		args = process.Args[2:]
	}

	ctx, err := Open(id)
	if err != nil {
		exception.New("pipes.%s not found", 404, id).Throw()
	}

	// merge the global data
	if process.Global != nil {
		merge := map[string]any{}
		global := process.Global
		for k, v := range global {
			merge[k] = v
		}

		if data != nil {
			for k, v := range data {
				merge[k] = v
			}
		}
		data = merge
	}

	return ctx.
		WithGlobal(data).
		WithSid(process.Sid).
		Resume(id, args...)
}

// processClose process the close pipe.close <id>
func processClose(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	Close(id)
	return nil
}
