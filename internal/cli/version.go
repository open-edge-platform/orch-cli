// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"runtime"
)

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Get catalog CLI version",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Printf("catalog version 0.12.0 %s\n", runtime.GOARCH)
			return nil
		},
	}
}
