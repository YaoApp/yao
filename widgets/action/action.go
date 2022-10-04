package action

// NewProcess create a new process
func NewProcess() *Process {
	return &Process{
		Default: []interface{}{},
	}
}

// ProcessOf create of get a process
func ProcessOf(p *Process) *Process {
	if p == nil {
		p = NewProcess()
	}
	return p
}
