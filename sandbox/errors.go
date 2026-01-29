package sandbox

import "errors"

var (
	// ErrTooManyContainers is returned when the maximum number of containers is reached
	ErrTooManyContainers = errors.New("sandbox: too many running containers, please try again later")

	// ErrContainerNotFound is returned when a container is not found
	ErrContainerNotFound = errors.New("sandbox: container not found")

	// ErrDockerNotAvailable is returned when Docker is not available
	ErrDockerNotAvailable = errors.New("sandbox: Docker not available")

	// ErrContainerNotRunning is returned when trying to execute on a non-running container
	ErrContainerNotRunning = errors.New("sandbox: container is not running")

	// ErrIPCSessionNotFound is returned when an IPC session is not found
	ErrIPCSessionNotFound = errors.New("sandbox: IPC session not found")

	// ErrToolNotAuthorized is returned when a tool is not authorized
	ErrToolNotAuthorized = errors.New("sandbox: tool not found or not authorized")
)
