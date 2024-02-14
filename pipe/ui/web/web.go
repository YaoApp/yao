package web

// Web the web UI
type Web struct{}

// Option the web option
type Option struct{}

// Render the Web UI
func (web *Web) Render(args []any, option Option) error {
	return nil
}
