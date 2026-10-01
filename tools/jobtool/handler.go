package jobtool

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/process"
)

//go:embed schema_job_start.json
var JobStartSchemaJSON []byte

//go:embed schema_job_list.json
var JobListSchemaJSON []byte

//go:embed schema_job_get.json
var JobGetSchemaJSON []byte

//go:embed schema_job_output.json
var JobOutputSchemaJSON []byte

//go:embed schema_job_wait.json
var JobWaitSchemaJSON []byte

//go:embed schema_job_stop.json
var JobStopSchemaJSON []byte

//go:embed schema_job_settled.json
var JobSettledSchemaJSON []byte

//go:embed schema_daemon_start.json
var DaemonStartSchemaJSON []byte

//go:embed schema_daemon_list.json
var DaemonListSchemaJSON []byte

//go:embed schema_daemon_status.json
var DaemonStatusSchemaJSON []byte

//go:embed schema_daemon_stop.json
var DaemonStopSchemaJSON []byte

//go:embed schema_daemon_restart.json
var DaemonRestartSchemaJSON []byte

//go:embed schema_daemon_port_ready.json
var DaemonPortReadySchemaJSON []byte

// stubLog prints the received tool call to stdout and returns the args as-is.
func stubLog(name string, proc *process.Process) interface{} {
	args := proc.ArgsMap(0)
	data, _ := json.Marshal(args)
	fmt.Printf("[jobtool] received tool=%s args=%s\n", name, string(data))
	return args
}

// JobStartHandler handles yao_job_start.
func JobStartHandler(proc *process.Process) interface{} { return stubLog("yao_job_start", proc) }

// JobListHandler handles yao_job_list.
func JobListHandler(proc *process.Process) interface{} { return stubLog("yao_job_list", proc) }

// JobGetHandler handles yao_job_get.
func JobGetHandler(proc *process.Process) interface{} { return stubLog("yao_job_get", proc) }

// JobOutputHandler handles yao_job_output.
func JobOutputHandler(proc *process.Process) interface{} { return stubLog("yao_job_output", proc) }

// JobWaitHandler handles yao_job_wait.
func JobWaitHandler(proc *process.Process) interface{} { return stubLog("yao_job_wait", proc) }

// JobStopHandler handles yao_job_stop.
func JobStopHandler(proc *process.Process) interface{} { return stubLog("yao_job_stop", proc) }

// JobSettledHandler handles yao_job_settled (async event from Monitor).
func JobSettledHandler(proc *process.Process) interface{} { return stubLog("yao_job_settled", proc) }

// DaemonStartHandler handles yao_daemon_start.
func DaemonStartHandler(proc *process.Process) interface{} {
	return stubLog("yao_daemon_start", proc)
}

// DaemonListHandler handles yao_daemon_list.
func DaemonListHandler(proc *process.Process) interface{} {
	return stubLog("yao_daemon_list", proc)
}

// DaemonStatusHandler handles yao_daemon_status.
func DaemonStatusHandler(proc *process.Process) interface{} {
	return stubLog("yao_daemon_status", proc)
}

// DaemonStopHandler handles yao_daemon_stop.
func DaemonStopHandler(proc *process.Process) interface{} {
	return stubLog("yao_daemon_stop", proc)
}

// DaemonRestartHandler handles yao_daemon_restart.
func DaemonRestartHandler(proc *process.Process) interface{} {
	return stubLog("yao_daemon_restart", proc)
}

// DaemonPortReadyHandler handles yao_daemon_port_ready (async event from Monitor).
func DaemonPortReadyHandler(proc *process.Process) interface{} {
	return stubLog("yao_daemon_port_ready", proc)
}
