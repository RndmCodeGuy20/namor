package main

import (
	"namor/cmd"
	"namor/internal/flags"
	"os"
)

func main() {
	rootCmd := cmd.NewRootCommand()
	flags.RegisterInitializationFlags(rootCmd)
	flags.RegisterDockerFlags(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
