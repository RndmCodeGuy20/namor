package runtime

import (
	"namor/pkg/container_types"
	"namor/pkg/errors"
)

type ContainerRuntimeError struct {
	*errors.BaseError
	Status container_types.ContainerRuntimeHealthStatus
}

func NewContainerRuntimeError(status container_types.ContainerRuntimeHealthStatus, message string) *ContainerRuntimeError {
	return &ContainerRuntimeError{
		BaseError: &errors.BaseError{
			Message: message,
			Code:    "CONTAINER_RUNTIME_ERROR",
			Details: map[string]interface{}{
				"status": status.String(),
			},
		},
		Status: status,
	}
}

type ContainerRuntime interface {
	IsDaemonActive() bool
	GetVersion() (string, error)
	StartHealthChecks()
	PullImage(image string) error
}
