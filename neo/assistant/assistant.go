package assistant

import "github.com/yaoapp/yao/neo/store"

// loadedAssistant the loaded assistant
var loadedAssistant = map[string]*Assistant{}

// LoadLocal create a new assistant from local
func LoadLocal(path string) *Assistant {
	return nil
}

// LoadZip create a new assistant from zip
func LoadZip(zip string) *Assistant {
	return nil
}

// LoadRemote create a new assistant from remote
func LoadRemote(url string) *Assistant {
	return nil
}

// LoadStore create a new assistant from store
func LoadStore(store store.Store) *Assistant {
	return nil
}
