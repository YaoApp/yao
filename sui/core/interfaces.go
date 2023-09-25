package core

// ITemplate is the interface for the ITemplate
type ITemplate interface {
	Get() error
	Save() error

	Pages() ([]IPage, error)
	Blocks() ([]IBlock, error)
	Components() ([]IComponent, error)

	Page(route string) (IPage, error)
	Block(name string) (IBlock, error)
	Component(name string) (IComponent, error)

	Styles() []string
	Locales() []string
	Themes() []string
}

// IPage is the interface for the page
type IPage interface {
	Load() error

	// Render()

	// Html()
	// Script()
	// Style()
	// Data()

	// Compile()
	// Locale()
}

// IBlock is the interface for the block
type IBlock interface {
	Compile() (string, error)
	Load() error
}

// IComponent is the interface for the component
type IComponent interface {
	Compile() (string, error)
	Load() error
}
