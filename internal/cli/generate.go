// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func getGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generic dynamic configuration files",
		Example: "orch-cli generate standalone-config --config-file <path-to-config-file>",
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
		getStandaloneConfigCommand(),
	)
	return cmd
}
