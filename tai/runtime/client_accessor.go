package runtime

import "github.com/docker/docker/client"

// dockerCliAccessor is implemented by runtime types that hold a Docker client.
type dockerCliAccessor interface {
	dockerClient() *client.Client
}

func (l *local) dockerClient() *client.Client         { return l.core.cli }
func (d *dockerSandbox) dockerClient() *client.Client { return d.core.cli }

// DockerCli extracts the underlying Docker SDK client from a Runtime.
// Returns nil if the Runtime is not Docker-based (e.g. K8s).
func DockerCli(rt Runtime) *client.Client {
	if a, ok := rt.(dockerCliAccessor); ok {
		return a.dockerClient()
	}
	return nil
}
