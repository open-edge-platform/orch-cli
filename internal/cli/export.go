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
		Use:               "export {<file-path>|<dir-path>} [flags]",
		Args:              cobra.ExactArgs(1),
		Short:             "Export catalog resources by saving them into a directory structure as YAML files",
		PersistentPreRunE: auth.CheckAuth,
		Example:           "orch-cli export /path/to/export --project my-project",
		RunE:              exportResources,
	}
	return cmd
}

func exportResources(cmd *cobra.Command, args []string) error {
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
