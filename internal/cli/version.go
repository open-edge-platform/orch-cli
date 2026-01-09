// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Version = "dev"

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Get Orchestrator CLI version",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Printf("Orchestrator CLI version %s %s\n", Version, runtime.GOARCH)

			if viper.GetString("version") != "" {
				fmt.Printf("Target Edge Orchestrator version %s\n", viper.GetString("version"))
			}
			return nil
		},
	}
}
