package runtime

import (
	"context"
	"time"
)

// Image manages container images on a runtime node.
type Image interface {
	Exists(ctx context.Context, ref string) (bool, error)
	Inspect(ctx context.Context, ref string) (*ImageMeta, error)
	Pull(ctx context.Context, ref string, opts PullOptions) (<-chan PullProgress, error)
	Remove(ctx context.Context, ref string, force bool) error
	List(ctx context.Context) ([]ImageInfo, error)
}

// ImageMeta holds static metadata extracted from a container image.
type ImageMeta struct {
	OS      string // "linux", "windows"
	Arch    string // "amd64", "arm64"
	Shell   string // preferred shell: "bash", "sh", "cmd.exe", "pwsh"
	WorkDir string // default working directory from Dockerfile WORKDIR
}

// PullOptions configures an image pull operation.
type PullOptions struct {
	Auth *RegistryAuth // nil = anonymous / public
}

// RegistryAuth holds credentials for a private container registry.
type RegistryAuth struct {
	Username string
	Password string
	Server   string // e.g. "ghcr.io", "registry.example.com"
}

// PullProgress reports real-time progress of an image pull.
type PullProgress struct {
	Status  string // "Pulling fs layer", "Downloading", "Extracting", "Pull complete", etc.
	Layer   string // layer digest / short ID
	Current int64  // bytes completed
	Total   int64  // bytes total (0 if unknown)
	Error   string // non-empty on failure
}

// ImageInfo describes a local image.
type ImageInfo struct {
	ID      string
	Tags    []string
	Size    int64
	Created time.Time
}
