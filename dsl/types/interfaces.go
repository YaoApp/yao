package types

import "context"

// DSL interface
type DSL interface {
	Inspect(ctx context.Context, id string) (*Info, error)        // Inspect DSL
	Path(ctx context.Context, id string) (string, error)          // Get Path by id, ( If the DSL is saved as file, return the file path )
	Source(ctx context.Context, id string) (string, error)        // Get Source by id
	List(ctx context.Context, opts *ListOptions) ([]*Info, error) // List All DSLs including unloaded/error DSLs
	Exists(ctx context.Context, id string) (bool, error)          // Check if the DSL exists

	// DSL Operations
	Create(ctx context.Context, options *CreateOptions) error                  // Create DSL, Create will unload the DSL first, then create the DSL to DB
	Update(ctx context.Context, options *UpdateOptions) error                  // Update DSL, Update will unload the DSL first, then update the DSL, if update info only, will not unload the DSL
	Delete(ctx context.Context, id string, unloadOptions ...interface{}) error // Delete DSL, Delete will unload the DSL first, then delete the DSL file

	// Load manager
	Load(ctx context.Context, id string, options interface{}) error      // Load DSL, Load will unload the DSL first, then load the DSL from DB or file system
	Reload(ctx context.Context, id string, options interface{}) error    // Reload DSL, Reload will unload the DSL first, then reload the DSL from DB or file system
	Unload(ctx context.Context, id string, options ...interface{}) error // Unload DSL, Unload will unload the DSL from memory

	// Execute
	Execute(ctx context.Context, method string, args ...any) (any, error) // Execute DSL (Some DSLs can be executed)

	// Validate
	Validate(ctx context.Context, source string) (bool, []LintMessage) // Validate DSL, Validate will validate the DSL from source
}

// Manager interface
type Manager interface {
	// Get all loaded DSLs
	Loaded(ctx context.Context) (map[string]*Info, error) // Get all loaded DSLs

	// Load DSL, Load will unload the DSL first, then load the DSL from DB or file system
	Load(ctx context.Context, id string, options interface{}) error

	// Unload DSL, Unload will unload the DSL from memory
	Unload(ctx context.Context, id string, options interface{}) error

	// Reload DSL, Reload will unload the DSL first, then reload the DSL from DB or file system
	Reload(ctx context.Context, id string, options interface{}) error

	// Validate DSL, Validate will validate the DSL from source
	Validate(ctx context.Context, source string) (bool, []LintMessage)

	// Execute DSL (Some DSLs can be executed)
	Execute(ctx context.Context, method string, args ...any) (any, error)
}
