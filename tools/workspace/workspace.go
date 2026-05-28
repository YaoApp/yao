package workspace

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	ws "github.com/yaoapp/yao/workspace"
)

func resolveOwner(auth *process.AuthorizedInfo) string {
	if auth != nil && auth.TeamID != "" {
		return auth.TeamID
	}
	if auth != nil {
		return auth.UserID
	}
	return ""
}

func resolveAndCheck(proc *process.Process, id string) (*ws.Workspace, error) {
	m := ws.M()
	if m == nil {
		return nil, fmt.Errorf("workspace service not available")
	}

	w, err := m.Get(proc.Context, id)
	if err != nil {
		if err == ws.ErrNotFound {
			return nil, fmt.Errorf("workspace not found")
		}
		return nil, err
	}

	owner := resolveOwner(proc.Authorized)
	if owner != "" && w.Owner != "" && w.Owner != owner {
		return nil, fmt.Errorf("no permission to access this workspace")
	}
	return w, nil
}
