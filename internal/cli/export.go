// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"
)

func getExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "export",
		Short:             "Export resources from the orchestrator",
		PersistentPreRunE: auth.CheckAuth,
		Example:           "orch-cli export deployment-package wordpress 0.1.1",
	}
	cmd.AddCommand(
		getExportDeploymentPackageCommand(),
	)
	return cmd
}
