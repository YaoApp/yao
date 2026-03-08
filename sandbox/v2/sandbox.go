package sandbox

var mgr *Manager

// Init initializes the global sandbox Manager.
// Node discovery is handled by the tai/registry; no configuration is needed.
func Init() {
	mgr = newManager()
}

// M returns the global Manager. Panics if Init was not called.
func M() *Manager {
	if mgr == nil {
		panic("sandbox.Init not called")
	}
	return mgr
}
