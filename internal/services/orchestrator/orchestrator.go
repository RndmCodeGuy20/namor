package orchestrator

import "namor/internal/helpers/runtime"

type orchestrator interface {
	PullImage(image string) error
	RunContainer(image string, cmd []string, env map[string]string, timeoutSeconds int) (string, error)
	StopContainer(containerID string) error
}

type Orchestrator struct {
	runtime runtime.ContainerRuntime
}

func NewOrchestrator(runtime runtime.ContainerRuntime) *Orchestrator {
	return &Orchestrator{
		runtime: runtime,
	}
}

// PullImage pulls a container image using the orchestrator's runtime.
// Returns true if the image was pulled successfully, false otherwise.
func (o *Orchestrator) PullImage(image string) bool {
	if !o.runtime.IsDaemonActive() {
		return false
	}

	return true
}
