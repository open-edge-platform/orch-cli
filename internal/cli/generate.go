// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/spf13/cobra"
)

func getGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generic dynamic configuration files",
		Example: "orch-cli generate standalone-config --file config-file",
	}
	cmd.AddCommand(
		getStandaloneConfigCommand(),
	)
	return cmd
}
