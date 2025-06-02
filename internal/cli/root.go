// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"

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

	catalogEndpoint = "api-endpoint"
	debugHeaders    = "debug-headers"
	project         = "project"

	// Default for dev deployment
	catalogDefaultEndpoint = "https://api.kind.internal/"

	deploymentEndpoint = "deployment-endpoint"

	// Default for dev deployment
	deploymentDefaultEndpoint = "https://api.kind.internal/"
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
		fmt.Println(err)
		os.Exit(1)
	}
}

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "orch-cli {create, get, set, list, delete, version} <resource> [flags]",
		Short:         "Orch-cli Command Line Interface",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Set some factory defaults as a fallback
	viper.SetDefault(catalogEndpoint, catalogDefaultEndpoint)
	viper.SetDefault(deploymentEndpoint, deploymentDefaultEndpoint)
	viper.SetDefault(debugHeaders, false)
	viper.SetDefault("verbose", false)
	viper.SetDefault(project, "")

	// Setup global persistent flags for endpoint addresses of various services
	rootCmd.PersistentFlags().String(catalogEndpoint, viper.GetString(catalogEndpoint), "API Service Endpoint")
	rootCmd.PersistentFlags().String(deploymentEndpoint, viper.GetString(deploymentEndpoint), "Deployment Service Endpoint")
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
		getWatchCommand(),
		getUploadCommand(),
		getLoginCommand(),
		getLogoutCommand(),
		getExportCommand(),
		getRegisterCommand(),
		getDeauthorizeCommand(),
		getWipeProjectCommand(),
		versionCommand(),
	)
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
