package flags

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const DockerMinApiVersion = "1.25"

func RegisterDockerFlags(cmd *cobra.Command) {
	persistentFlags := cmd.PersistentFlags()
	persistentFlags.StringP("docker_host", "D", envString("DOCKER_HOST"), "Docker daemon socket to connect to (default is unix:///var/run/docker.sock or npipe:////./pipe/docker_engine)")
	persistentFlags.BoolP("docker_tls_verify", "T", envBool("DOCKER_TLS_VERIFY"), "Use TLS and verify the remote")
	persistentFlags.StringP("docker_api_version", "A", envString("DOCKER_API_VERSION"), "Set the API version to use when communicating with the daemon")
}

func RegisterInitializationFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringP("config-file", "c", "config.yaml", "Path to the configuration file")
	flags.String("secret", "", "Secret token to validate incoming webhooks")
}

func envString(key string) string {
	viper.MustBindEnv(key)
	return viper.GetString(key)
}

func envStringSlice(key string) []string {
	viper.MustBindEnv(key)
	return viper.GetStringSlice(key)
}

func envInt(key string) int {
	viper.MustBindEnv(key)
	return viper.GetInt(key)
}

func envBool(key string) bool {
	viper.MustBindEnv(key)
	return viper.GetBool(key)
}

func envDuration(key string) time.Duration {
	viper.MustBindEnv(key)
	return viper.GetDuration(key)
}

func SetDefaults() {
	viper.AutomaticEnv()
	viper.SetDefault("DOCKER_HOST", "unix:///var/run/docker.sock")
	viper.SetDefault("DOCKER_TLS_VERIFY", false)
	viper.SetDefault("DOCKER_API_VERSION", DockerMinApiVersion)
}

func EnvFlags(cmd *cobra.Command) error {
	persistentFlags := cmd.PersistentFlags()

	if _, err := persistentFlags.GetString("docker_host"); err != nil {
		return err
	}

	if _, err := persistentFlags.GetBool("docker_tls_verify"); err != nil {
		return err
	}

	if _, err := persistentFlags.GetString("docker_api_version"); err != nil {
		return err
	}

	return nil
}

func GetDockerHost(cmd *cobra.Command) (string, error) {
	persistentFlags := cmd.PersistentFlags()
	return persistentFlags.GetString("docker_host")
}

func ReadFlags(cmd *cobra.Command) (string, bool, float32) {
	persistentFlags := cmd.PersistentFlags()

	var dockerHost string
	var dockerTLSVerify bool
	var dockerAPIVersion string

	dockerHost, _ = persistentFlags.GetString("docker_host")
	dockerTLSVerify, _ = persistentFlags.GetBool("docker_tls_verify")
	dockerAPIVersion, _ = persistentFlags.GetString("docker_api_version")

	return dockerHost, dockerTLSVerify, parseAPIVersion(dockerAPIVersion)
}

func parseAPIVersion(version string) float32 {
	var major, minor int
	_, err := fmt.Sscanf(version, "%d.%d", &major, &minor)
	if err != nil {
		return 0.0
	}
	return float32(major) + float32(minor)/10.0
}
