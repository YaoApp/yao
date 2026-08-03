package workspace_config

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	ws "github.com/yaoapp/yao/workspace"
)

func resolveAndCheck(proc *process.Process, id string) error {
	m := ws.M()
	if m == nil {
		return fmt.Errorf("workspace service not available")
	}

	w, err := m.Get(proc.Context, id)
	if err != nil {
		if err == ws.ErrNotFound {
			return fmt.Errorf("workspace not found")
		}
		return err
	}

	owner := resolveOwner(proc.Authorized)
	if owner != "" && w.Owner != "" && w.Owner != owner {
		return fmt.Errorf("no permission to access this workspace")
	}
	return nil
}

func resolveOwner(auth *process.AuthorizedInfo) string {
	if auth != nil && auth.TeamID != "" {
		return auth.TeamID
	}
	if auth != nil {
		return auth.UserID
	}
	return ""
}
