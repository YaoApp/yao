package compute

const (
	// View View component
	View uint8 = iota
	// Edit Edit component
	Edit
	// Filter Filter component
	Filter
)

// Computable with computes
type Computable struct {
	Computes *Maps
}

// Maps compute mapping
type Maps struct {
	Edit   map[string][]Unit
	View   map[string][]Unit
	Filter map[string][]Unit
}

// Unit the compute unit
type Unit struct {
	Name string // index
	Kind uint8  // Type
}
