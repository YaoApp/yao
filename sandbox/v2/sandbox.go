package sandbox

var mgr *Manager

// Init initializes the global sandbox Manager.
// Config contains pool definitions. At least one Pool entry is required.
// Pass empty Pool list to disable sandbox (methods return ErrNotAvailable).
func Init(cfg Config) error {
	m, err := newManager(cfg)
	if err != nil {
		return err
	}
	mgr = m
	return nil
}

// M returns the global Manager. Panics if Init was not called.
func M() *Manager {
	if mgr == nil {
		panic("sandbox.Init not called")
	}
	return mgr
}
