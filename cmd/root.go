package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	VERSION string
	// Global application context for graceful shutdown
	appContext context.Context
	cancelFunc context.CancelFunc
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "terragrunt-atlantis-config",
	Short:        "Generates Atlantis Config for Terragrunt projects",
	Long:         "Generates Atlantis Config for Terragrunt projects",
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	VERSION = version

	// Setup graceful shutdown with signal handling
	appContext, cancelFunc = context.WithCancel(context.Background())

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancelFunc()    // Cancel the context to signal shutdown
		cleanupCaches() // Clean up all caches
	}()

	// Setup cleanup on exit for graceful shutdown
	defer func() {
		cancelFunc()
		cleanupCaches()
	}()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
