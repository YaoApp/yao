package sandbox

import "time"

// ResetForTest resets the global manager for testing purposes.
func ResetForTest() {
	mgr = nil
}

// ExportApplyExecOptions applies ExecOption functions and returns
// the resulting config fields for black-box testing.
func ExportApplyExecOptions(opts ...ExecOption) (workDir string, env map[string]string, timeout int64, stdin []byte, maxOutput int64) {
	var c execConfig
	for _, o := range opts {
		o(&c)
	}
	return c.WorkDir, c.Env, int64(c.Timeout), c.Stdin, c.MaxOutputBytes
}

// ExportApplyAttachOptions applies AttachOption functions and returns
// the resulting config fields for black-box testing.
func ExportApplyAttachOptions(opts ...AttachOption) (protocol, path string, headers map[string]string) {
	var c attachConfig
	for _, o := range opts {
		o(&c)
	}
	return c.Protocol, c.Path, c.Headers
}

// ExportDefaultWorkspaceName exposes defaultWorkspaceName for testing.
var ExportDefaultWorkspaceName = defaultWorkspaceName

// ExportSystemInfoFromLabels exposes systemInfoFromLabels for testing.
var ExportSystemInfoFromLabels = systemInfoFromLabels

// ExportNewBoxForTest creates a minimal Box for unit testing without
// requiring a Manager or Tai connection.
func ExportNewBoxForTest(id, owner, containerID, nodeID, workDir, image string,
	policy LifecyclePolicy, labels map[string]string, displayName string,
	sys SystemInfo, idleTimeout, maxLifetime, stopTimeout time.Duration,
) *Box {
	b := &Box{
		id:           id,
		owner:        owner,
		containerID:  containerID,
		nodeID:       nodeID,
		workDir:      workDir,
		image:        image,
		policy:       policy,
		labels:       labels,
		displayName:  displayName,
		system:       sys,
		idleTimeoutD: idleTimeout,
		maxLifetimeD: maxLifetime,
		stopTimeoutD: stopTimeout,
		createdAt:    time.Now(),
	}
	return b
}

// ExportNewHostForTest creates a minimal Host for unit testing.
func ExportNewHostForTest(nodeID string, sys SystemInfo) *Host {
	return &Host{
		nodeID: nodeID,
		system: sys,
	}
}

// ExportWatcherName returns the sandbox watcher name.
func ExportWatcherName() string {
	w := &sandboxWatcher{}
	return w.Name()
}

// ExportWatcherInterval returns the sandbox watcher interval.
func ExportWatcherInterval() time.Duration {
	w := &sandboxWatcher{}
	return w.Interval()
}
