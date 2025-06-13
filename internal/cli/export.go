// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/loader"
	"github.com/spf13/cobra"
)

func getExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "export",
		Short:             "Export resoources from the orchestrator",
		PersistentPreRunE: auth.CheckAuth,
		Example:           "orch-cli export deployment-package wordpress 0.1.1",
	}
	cmd.AddCommand(
		getExportDeploymentPackageCommand(),
		getExportCatalogCommand(),
	)
	return cmd
}

/*
 * getExportCatalogCommand is a command that exports all catalog resources.
 *
 * TODO: Evaluate whether this is useful.
 */

func getExportCatalogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-catalog-resources <dest-dir>",
		Short: "Export the entire catalog as a set of individual yaml files",
		Args:  cobra.ExactArgs(1),
		RunE:  exportCatalog,
	}
	return cmd
}

func exportCatalog(cmd *cobra.Command, args []string) error {
	_, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}
	verbose, _ := cmd.Flags().GetBool("verbose")
	exporter := loader.NewExporter()
	err = exporter.ExportCatalogItems(context.Background(), catalogClient, projectName, args[0])
	if err != nil {
		return err
	}
	if verbose {
		for _, line := range exporter.GetOutput() {
			fmt.Printf("%s", line)
		}
	}
	return nil
}
