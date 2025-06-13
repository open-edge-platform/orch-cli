// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/loader"
	"github.com/spf13/cobra"
)

func getUploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "upload {<file-path>|<dir-path>} [flags]",
		Aliases:           []string{"load"},
		Args:              cobra.ExactArgs(1),
		Short:             "Create catalog resources by uploading YAML files",
		PersistentPreRunE: auth.CheckAuth,
		Example:           "orch-cli upload /path/to/resource.yaml --project some-project",
		RunE:              uploadResources,
	}
	return cmd
}

func uploadResources(cmd *cobra.Command, args []string) error {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return err
	}

	projectUUID, err := getProjectName(cmd)
	if err != nil {
		return err
	}

	loader := loader.NewLoader(serverAddress, projectUUID)
	return loader.LoadResources(context.Background(), "", args)
}
