package container_types

type ContainerRuntimeHealthStatus int

const (
	Healthy = ContainerRuntimeHealthStatus(iota)
	Unhealthy
	Degraded
)

func (s ContainerRuntimeHealthStatus) String() string {
	return [...]string{"Health", "Unhealthy", "Degraded"}[s]
}
