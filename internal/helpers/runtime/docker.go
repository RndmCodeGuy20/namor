package runtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"namor/pkg/container_types"
	"namor/pkg/utils"
	"namor/pkg/utils/container"
	"os"
	"sync"
	"time"

	api "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/plugins"
	"github.com/moby/term"
	"go.uber.org/zap"
)

var (
	dockerRuntimeInstance *DockerRuntime
	once                  sync.Once
)

// DockerRuntime implements ContainerRuntime for Docker.
type DockerRuntime struct {
	client         *client.Client
	logger         *utils.ServiceLogger
	circuitBreaker *container.RuntimeCircuitBreaker

	healthCheckCtx    context.Context
	healthCheckCancel context.CancelFunc

	minVersion string

	mu            sync.RWMutex
	currentStatus container_types.ContainerRuntimeHealthStatus

	initialized bool
}

func GetDockerRuntime(host, minVersion string) (*DockerRuntime, error) {
	once.Do(func() {
		logger := utils.NewServiceLogger("docker_runtime")
		circuitBreaker := container.NewCircuitBreaker()

		logger.Info("Checking Docker runtime connectivity on host: " + host)

		dockerClient, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())

		if err != nil {
			logger.Error("Failed to create Docker client", zap.String("reason", err.Error()))
			return
		}

		dockerRuntimeInstance = &DockerRuntime{
			client:         dockerClient,
			logger:         logger,
			circuitBreaker: circuitBreaker,
			minVersion:     minVersion,
			currentStatus:  container_types.Unhealthy, // Start as unhealthy until proven otherwise
			initialized:    true,
		}

		dockerRuntimeInstance.circuitBreaker.OnStateChange = func(from, to container.CircuitBreakerState) {
			dockerRuntimeInstance.logger.Warn("Docker runtime circuit breaker state changed", zap.String("from", from.String()), zap.String("to", to.String()))
		}
		dockerRuntimeInstance.circuitBreaker.OnFailure = func(err error) {
			dockerRuntimeInstance.logger.Error("Docker runtime operation failed", zap.String("reason", err.Error()))
		}

		logger.Info("Initializing docker runtime with default settings")
	})

	if !dockerRuntimeInstance.initialized {
		return nil, NewContainerRuntimeError(container_types.Unhealthy, "Docker runtime initialization failed")
	}

	return dockerRuntimeInstance, nil
}

func (dr *DockerRuntime) IsDaemonActive() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ping, err := dr.client.Ping(ctx)
	if err != nil {
		return false
	}

	dr.logger.Info("Docker daemon is accessible.", zap.String("api_version", ping.APIVersion))

	return true
}

func (dr *DockerRuntime) GetVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	version, err := dr.client.ServerVersion(ctx)
	if err != nil {
		return "", err
	}

	return version.Version, nil
}

func (dr *DockerRuntime) StartHealthChecks() {
	dr.mu.Lock()
	if dr.healthCheckCtx != nil {
		dr.mu.Unlock()
		// Already running
		return
	}
	dr.healthCheckCtx, dr.healthCheckCancel = context.WithCancel(context.Background())
	dr.mu.Unlock()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				dr.performHealthCheck()
			case <-dr.healthCheckCtx.Done():
				return
			}
		}
	}()

	// Initial check
	dr.performHealthCheck()
}

func (dr *DockerRuntime) StopHealthChecks() {
	dr.mu.Lock()
	if dr.healthCheckCancel != nil {
		dr.healthCheckCancel()
		dr.healthCheckCtx = nil
		dr.healthCheckCancel = nil
	}
	dr.mu.Unlock()
}

func (dr *DockerRuntime) PullImage(image, username, token, serverAddress string) error {
	dr.logger.Debug("Pulling image: " + image)

	// Use circuit breaker to wrap the pull operation
	err := dr.circuitBreaker.Execute(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		dr.logger.Debug("Pulling image: "+image, zap.String("server_address", serverAddress), zap.String("username", username), zap.String("token", token))

		authConfig := registry.AuthConfig{
			Username:      username,
			Password:      token,
			ServerAddress: serverAddress,
		}

		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return fmt.Errorf("failed to encode auth config: %w", err)
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)

		out, err := dr.client.ImagePull(ctx, image, api.PullOptions{RegistryAuth: authStr})
		if err != nil {
			// get error type
			if plugins.IsNotFound(err) {
				return NewContainerRuntimeError(dr.currentStatus, fmt.Sprintf("image %s not found", image))
			}
			return err
		}
		defer func(out io.ReadCloser) {
			err := out.Close()
			if err != nil {
				return
			}
		}(out)

		// Get file descriptor for stdout
		fd, isTerminal := term.GetFdInfo(os.Stdout)

		// Stream JSON messages into human-readable output
		err = jsonmessage.DisplayJSONMessagesStream(out, os.Stdout, fd, isTerminal, nil)
		if err != nil {
			return fmt.Errorf("failed to display image pull stream: %w", err)
		}

		return nil
	})

	dr.logger.Debug("Image pull completed.")

	if err != nil {
		return NewContainerRuntimeError(container_types.Unhealthy, "Failed to pull image: "+err.Error())
	}

	return nil
}

func (dr *DockerRuntime) CreateContainer(image, containerName string, ports, env, volumes []string) error {
	dr.logger.Debug("Creating container: " + containerName)

	err := dr.circuitBreaker.Execute(func() error {
		// configure container creation options
		// var hostConfig *docker_container.HostConfig

		// portBinding := nat.PortMap{}
		// for port := range ports {
		// 	portMapping, err := nat.ParsePortSpec(ports[port])
		// 	if err != nil {
		// 		return fmt.Errorf("invalid port mapping %s: %w", ports[port], err)
		// 	}

		// }

		return nil
	})

	if err != nil {
		return NewContainerRuntimeError(dr.currentStatus, "Failed to create container: "+err.Error())
	}

	return nil
}

func (dr *DockerRuntime) performHealthCheck() {
	err := dr.circuitBreaker.Execute(func() error {
		if !dr.IsDaemonActive() {
			return NewContainerRuntimeError(container_types.Unhealthy, "Docker daemon is not active")
		}

		version, err := dr.GetVersion()
		if err != nil {
			return NewContainerRuntimeError(container_types.Unhealthy, "Failed to get Docker version: "+err.Error())
		}

		dr.logger.Info("Docker version: " + version)

		// Here you could add version comparison logic if needed

		return nil
	})

	dr.mu.Lock()
	defer dr.mu.Unlock()

	if err != nil {
		dr.currentStatus = container_types.Unhealthy
		dr.logger.Error("Docker runtime health check failed", zap.String("reason", err.Error()))
	} else {
		dr.currentStatus = container_types.Healthy
		dr.logger.Info("Docker runtime is healthy")
	}
}

func (dr *DockerRuntime) GetHealthStatus() container_types.ContainerRuntimeHealthStatus {
	dr.mu.RLock()
	defer dr.mu.RUnlock()
	return dr.currentStatus
}
