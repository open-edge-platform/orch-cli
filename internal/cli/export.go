// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"
)

func getExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "export",
		Short:             "Export resources from the orchestrator",
		PersistentPreRunE: auth.CheckAuth,
		Example:           "orch-cli export deployment-package wordpress 0.1.1",
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
		getExportDeploymentPackageCommand(),
	)
	return cmd
}
