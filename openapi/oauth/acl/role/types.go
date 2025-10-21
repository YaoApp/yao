package role

import (
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Manager is the role manager
type Manager struct {
	cache    store.Store
	provider types.UserProvider
}
