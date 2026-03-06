package sandbox

import "github.com/docker/docker/client"

// dockerCliAccessor is implemented by sandbox types that hold a Docker client.
type dockerCliAccessor interface {
	dockerClient() *client.Client
}

func (l *local) dockerClient() *client.Client         { return l.core.cli }
func (d *dockerSandbox) dockerClient() *client.Client { return d.core.cli }

// DockerCli extracts the underlying Docker SDK client from a Sandbox.
// Returns nil if the Sandbox is not Docker-based (e.g. K8s).
func DockerCli(sb Sandbox) *client.Client {
	if a, ok := sb.(dockerCliAccessor); ok {
		return a.dockerClient()
	}
	return nil
}
