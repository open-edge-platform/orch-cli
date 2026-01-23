// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/atomix/dazl"
	clilib "github.com/open-edge-platform/orch-library/go/pkg/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

var log = dazl.GetLogger()

// Default value for the catalog service REST end-point.
const (
	CLIName = "orch-cli"

	apiEndpoint  = "api-endpoint"
	debugHeaders = "debug-headers"
	project      = "project"

	// Default for dev deployment
	apiDefaultEndpoint = "https://api.kind.internal/"
)

// init initializes the command line
func init() {
	// Set the config directory relative path
	clilib.SetConfigDir("." + CLIName)

	// Initialize the config name
	clilib.InitConfig(CLIName)

	// Pre-create the config
	_ = clilib.CreateConfig(false)
}

// Init is a hook called after cobra initialization
func Init() {
	// noop for now
}

// Execute is tha main entry point for the command-line execution.
func Execute() {
	rootCmd := getRootCmd()
	if err := rootCmd.Execute(); err != nil {
		// Check if this is an unknown command error for a disabled command
		if errStr := err.Error(); strings.Contains(errStr, "unknown command") {
			// Extract the command name from the error
			// Error format: unknown command "wipe" for "orch-cli"
			if start := strings.Index(errStr, "\""); start != -1 {
				if end := strings.Index(errStr[start+1:], "\""); end != -1 {
					cmdName := errStr[start+1 : start+1+end]
					if isCommandDisabledWithParent(rootCmd, cmdName) {
						fmt.Fprintf(os.Stderr, "Error: command %q is disabled in the current Edge Orchestrator configuration\n", cmdName)
						os.Exit(1)
					}
				}
			}
			// It's a truly unknown command - print the error with help suggestion
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			fmt.Fprintf(os.Stderr, "Run '%s --help' for usage.\n", rootCmd.CommandPath())
		} else {
			// Other errors - print them
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "orch-cli {create, get, set, list, delete, version} <resource> [flags]",
		Short:         "Orch-cli Command Line Interface",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set some factory defaults as a fallback
	viper.SetDefault(apiEndpoint, apiDefaultEndpoint)
	viper.SetDefault(debugHeaders, false)
	viper.SetDefault("verbose", false)
	viper.SetDefault(project, "")

	// Setup global persistent flags for endpoint addresses of various services
	rootCmd.PersistentFlags().String(apiEndpoint, viper.GetString(apiEndpoint), "API Service Endpoint")
	rootCmd.PersistentFlags().Bool(debugHeaders, viper.GetBool(debugHeaders), "emit debug-style headers separating columns via '|' character")
	rootCmd.PersistentFlags().StringP(project, "p", viper.GetString(project), "Active project name")

	// Setup global persistent flag for verbose output
	var Verbose bool
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", viper.GetBool("verbose"), "produce verbose output")
	var NoAuth bool
	rootCmd.PersistentFlags().BoolVarP(&NoAuth, "noauth", "n", viper.GetBool("noauth"), "use without authentication checks")

	rootCmd.AddCommand(
		clilib.GetConfigCommand(),
		getCreateCommand(),
		getListCommand(),
		getGetCommand(),
		getSetCommand(),
		getDeleteCommand(),

		getLoginCommand(),
		getLogoutCommand(),

		versionCommand(),

		getGenerateCommand(),
	)

	addCommandIfFeatureEnabled(rootCmd, getDeauthorizeCommand(), OnboardingFeature)

	addCommandIfFeatureEnabled(rootCmd, getUpdateCommand(), Day2Feature)

	addCommandIfFeatureEnabled(rootCmd, getWipeProjectCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(rootCmd, getImportCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(rootCmd, getUploadCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(rootCmd, getUpgradeCommand(), AppOrchFeature)
	addCommandIfFeatureEnabled(rootCmd, getExportCommand(), AppOrchFeature)

	return rootCmd
}

// GenerateDocs generates markdown documentation for the suite of catalog service CLI commands.
func GenerateDocs() {
	cmd := getRootCmd()
	err := doc.GenMarkdownTree(cmd, "docs/cli")
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
