package sandboxv2

import (
	"fmt"
	"sync"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

var (
	mu      sync.RWMutex
	runners = map[string]func() types.Runner{}
)

// Register adds a runner factory to the global registry.
// Typically called from init() in the runner's package.
func Register(name string, factory func() types.Runner) {
	mu.Lock()
	defer mu.Unlock()
	runners[name] = factory
}

// Get creates a new Runner instance from the registry.
func Get(name string) (types.Runner, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := runners[name]
	if !ok {
		return nil, fmt.Errorf("sandbox runner %q not registered", name)
	}
	return factory(), nil
}
