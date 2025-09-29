package cmd

import (
	"fmt"
	"namor/internal/config"
	"namor/internal/flags"
	"namor/internal/helpers/runtime"
	"namor/pkg/errors"
	"namor/pkg/utils"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	Version = "1.0.0"
	Commit  = "abc1234"
	Date    = "2024-06-01"
	BuiltBy = "developer"
)

var logger *utils.ServiceLogger

func NewRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "namor",
		Short: "Lightweight webhook based automated deployment tool",
		Long:  `Namor is a lightweight, no BS, webhook based automated deployment tool.`,
		Args:  cobra.ArbitraryArgs,
		Run:   Run,
	}
}

func initialize(cmd *cobra.Command) {
	flags.SetDefaults()
	flags.RegisterDockerFlags(cmd)
	flags.RegisterInitializationFlags(cmd)
}

func Run(cmd *cobra.Command, args []string) {
	logger = utils.NewServiceLogger("boot")
	defer logger.Close()
	if err := Execute(cmd, args); err != nil {
		logger.Error(fmt.Sprintf("Namor failed to start ::: %s", userFriendlyError(err)))
		logger.Fatal("Exiting with code (1)...")
	}
}

// Helper to extract a readable error message
func userFriendlyError(err error) string {
	// If your error type has more details, extract them here
	return err.Error()
}

func Execute(cmd *cobra.Command, args []string) error {
	flags.SetDefaults()
	if err := flags.EnvFlags(cmd); err != nil {
		return err
	}

	configPath, _ := cmd.Flags().GetString("config-file")
	secret, _ := cmd.Flags().GetString("secret")

	if secret == "" {
		return errors.NewInitializationError("Secret must be provided via command line flag or environment variable", nil)
	}

	logger.Info("Starting Namor...")

	var namorConfig *config.NamorConfig
	var err error

	if configPath != "" {
		logger.Info("Loading configuration from specified path: " + configPath)
		namorConfig, err = config.LoadConfig(configPath)

		if err != nil {
			return err
		}
	} else {
		namorConfig, err = config.LoadConfig(filepath.Join(utils.GetWorkingDir(), "config.yaml"))
	}

	if err != nil {
		return err
	}

	// pass the config to the webhook service and the docker service
	logger.Info("Namor configuration loaded")

	dockerRuntime, err := runtime.GetDockerRuntime(namorConfig.Host, flags.DockerMinApiVersion)
	if err != nil {
		return err
	}

	version, err := dockerRuntime.GetVersion()
	if err != nil {
		return err
	}

	logger.Sugar().Infof("Docker runtime version: %s", version)

	service := namorConfig.Services["conduit"]
	baseRegistry, username := strings.Split(service.Registry, "/")[0], strings.Split(service.Registry, "/")[1]

	if service.RequiresAuth && (baseRegistry == "" || username == "") {
		return errors.NewInitializationError("Registry and username must be provided for authenticated images", nil)
	}
	err = dockerRuntime.PullImage(
		service.Registry+"/"+"conduit:0e25fd4",
		username,
		"",
		baseRegistry,
	)
	if err != nil {
		return err
	}

	logger.Debug("Docker image 'hello-world' pulled successfully")

	return nil
}
