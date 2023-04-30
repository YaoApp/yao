package conversation

// Setting the conversation config
type Setting struct {
	Connector string `json:"connector,omitempty"`
	Table     string `json:"table,omitempty"`
	MaxSize   int    `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	TTL       int    `json:"ttl,omitempty" yaml:"ttl,omitempty"`
}
