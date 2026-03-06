package workspace

import "errors"

var (
	ErrNotFound    = errors.New("workspace: not found")
	ErrNodeMissing = errors.New("workspace: node is required")
	ErrNodeOffline = errors.New("workspace: node is offline or not configured")
	ErrHasMounts   = errors.New("workspace: workspace has active container mounts")
)
