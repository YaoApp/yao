package runtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

// dockerImage implements Image using the Docker SDK.
// Shared by both local and docker (via Tai proxy) runtime modes.
type dockerImage struct {
	cli *client.Client
}

// NewDockerImage creates an Image backed by a Docker client.
func NewDockerImage(cli *client.Client) Image {
	return &dockerImage{cli: cli}
}

func (d *dockerImage) Exists(ctx context.Context, ref string) (bool, error) {
	_, _, err := d.cli.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("image inspect %q: %w", ref, err)
	}
	return true, nil
}

func (d *dockerImage) Inspect(ctx context.Context, ref string) (*ImageMeta, error) {
	inspect, _, err := d.cli.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("image inspect %q: %w", ref, err)
	}

	meta := &ImageMeta{
		OS:   inspect.Os,
		Arch: inspect.Architecture,
	}

	if inspect.Config != nil {
		meta.WorkDir = inspect.Config.WorkingDir

		if len(inspect.Config.Shell) > 0 {
			meta.Shell = inspect.Config.Shell[0]
		}
		if meta.Shell == "" {
			for _, e := range inspect.Config.Env {
				if strings.HasPrefix(e, "SHELL=") {
					meta.Shell = e[6:]
					break
				}
			}
		}
	}

	if meta.Shell == "" {
		if strings.EqualFold(meta.OS, "windows") {
			meta.Shell = "cmd.exe"
		} else {
			meta.Shell = "bash"
		}
	}

	return meta, nil
}

func (d *dockerImage) Pull(ctx context.Context, ref string, opts PullOptions) (<-chan PullProgress, error) {
	pullOpts := image.PullOptions{}
	if opts.Auth != nil {
		encoded, err := encodeAuth(opts.Auth)
		if err != nil {
			return nil, err
		}
		pullOpts.RegistryAuth = encoded
	}

	reader, err := d.cli.ImagePull(ctx, ref, pullOpts)
	if err != nil {
		return nil, fmt.Errorf("image pull %q: %w", ref, err)
	}

	ch := make(chan PullProgress, 32)
	go func() {
		defer close(ch)
		defer reader.Close()
		decodePullStream(reader, ch)
	}()
	return ch, nil
}

func (d *dockerImage) Remove(ctx context.Context, ref string, force bool) error {
	_, err := d.cli.ImageRemove(ctx, ref, image.RemoveOptions{Force: force, PruneChildren: true})
	if err != nil {
		return fmt.Errorf("image remove %q: %w", ref, err)
	}
	return nil
}

func (d *dockerImage) List(ctx context.Context) ([]ImageInfo, error) {
	imgs, err := d.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("image list: %w", err)
	}
	result := make([]ImageInfo, len(imgs))
	for i, img := range imgs {
		result[i] = ImageInfo{
			ID:      img.ID,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: time.Unix(img.Created, 0),
		}
	}
	return result, nil
}

// dockerPullEvent mirrors the JSON lines emitted by Docker's ImagePull stream.
type dockerPullEvent struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	Error string `json:"error"`
}

func decodePullStream(r io.Reader, ch chan<- PullProgress) {
	dec := json.NewDecoder(r)
	for {
		var ev dockerPullEvent
		if err := dec.Decode(&ev); err != nil {
			if err != io.EOF {
				ch <- PullProgress{Error: err.Error()}
			}
			return
		}
		p := PullProgress{
			Status:  ev.Status,
			Layer:   ev.ID,
			Current: ev.ProgressDetail.Current,
			Total:   ev.ProgressDetail.Total,
		}
		if ev.Error != "" {
			p.Error = ev.Error
		}
		ch <- p
	}
}

func encodeAuth(auth *RegistryAuth) (string, error) {
	cfg := registry.AuthConfig{
		Username:      auth.Username,
		Password:      auth.Password,
		ServerAddress: auth.Server,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("encode registry auth: %w", err)
	}
	return base64.URLEncoding.EncodeToString(data), nil
}
