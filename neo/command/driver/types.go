package driver

// Request the command request
type Request struct {
	ID  string
	Sid string
	Cid string
}

// Command the command struct
type Command struct {
	ID          string                   `json:"-" yaml:"-"`
	Use         string                   `json:"use,omitempty"`
	Name        string                   `json:"name,omitempty"`
	Description string                   `json:"description,omitempty"`
	Args        []map[string]interface{} `json:"args,omitempty"`
	Stack       string                   `json:"stack,omitempty"`
	Path        string                   `json:"path,omitempty"`
}
