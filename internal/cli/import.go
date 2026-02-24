// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"

	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
)

func getImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "import",
		Short:             "Create orchestrator resources by importing from an external source",
		PersistentPreRunE: auth.CheckAuth,
		Example:           "orch-cli import helm-chart oci:/path/to/chart:1.0.0 --project some-project",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				if isCommandDisabledWithParent(c, args[0]) {
					fmt.Fprintf(c.ErrOrStderr(), "Error: command %q is disabled in the current Edge Orchestrator configuration\n\n", args[0])
				} else {
					fmt.Fprintf(c.ErrOrStderr(), "Error: unknown command %q for %q\n\n", args[0], c.CommandPath())
				}
			}
			return c.Usage()
		},
	}
	cmd.AddCommand(
		getImportHelmChartCommand(),
	)
	return cmd
}

/*
 * getImportHelmCharCommand implements a command that imports helm charts into the catalog.
 *
 * As this is the only "import" subcommand, it's located here alongside the import command.
 * If more import commands are added in the future, they can be organized into a separate file.
 */

func getImportHelmChartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm-chart <chart-path> [flags]",
		Short: "Import a helm chart into the catalog",
		Args:  cobra.ExactArgs(1),
		RunE:  runImportHelmChartCommand,
	}
	cmd.Flags().StringP("values-file", "f", "", "filename for values.yaml")
	cmd.Flags().String("username", "", "OCI registry username")
	cmd.Flags().String("password", "", "OCI registry password / authentication token")
	cmd.Flags().Bool("include-auth", false, "Include authentication information in the imported chart")
	cmd.Flags().Bool("generate-default-values", false, "Generate default values for the chart")
	cmd.Flags().Bool("generate-default-parameters", false, "Generate default parameters for the chart")
	cmd.Flags().String("namespace", "", "Namespace to use for the imported chart")
	return cmd
}

func runImportHelmChartCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return processError(err)
	}

	chartValues := ""
	if valuesFile, err := cmd.Flags().GetString("values-file"); err != nil {
		return processError(err)
	} else if valuesFile != "" {
		chartValuesData, err := os.ReadFile(valuesFile)
		if err != nil {
			return processError(err)
		}
		chartValues = string(chartValuesData)
	}

	ociURL := args[0]
	resp, err := catalogClient.CatalogServiceImport(ctx, projectName,
		&catapi.CatalogServiceImportParams{
			Url:                       ociURL,
			Username:                  getFlag(cmd, "username"),
			AuthToken:                 getFlag(cmd, "password"),
			ChartValues:               &chartValues,
			IncludeAuth:               getBoolFlagOrDefault(cmd, "include-auth", nil),
			GenerateDefaultValues:     getBoolFlagOrDefault(cmd, "generate-default-values", nil),
			GenerateDefaultParameters: getBoolFlagOrDefault(cmd, "generate-default-parameters", nil),
			Namespace:                 getFlagOrDefault(cmd, "namespace", nil),
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	// Print the gRPC error message to the user as it might have insight into the failure.
	return checkResponseGRPC(resp, fmt.Sprintf("error while importing helm chart %s", ociURL))
}
