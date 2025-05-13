// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"
)

func getRegisterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "register",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Register host",
		PersistentPreRunE: auth.CheckAuth,
	}

	cmd.AddCommand(
		getRegisterHostCommand(),
	)
	return cmd
}

func getRegisterHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "Register a host",
		Args:  cobra.ExactArgs(1),
		RunE:  runRegisterHostCommand,
	}
	return cmd
}

func runRegisterHostCommand(cmd *cobra.Command, _ []string) error {
	return nil
}

func getListHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "List hosts",
		RunE:  runListHostCommand,
	}
	return cmd
}

func runListHostCommand(cmd *cobra.Command, _ []string) error {
	return nil
}

func getGetHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <name> [flags]",
		Short: "Get host",
		Args:  cobra.ExactArgs(1),
		RunE:  runGetHostCommand,
	}
	return cmd
}

func runGetHostCommand(cmd *cobra.Command, _ []string) error {
	return nil
}
