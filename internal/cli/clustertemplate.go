// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"

	coapi "github.com/open-edge-platform/cli/pkg/rest/cluster"
)

var clusterTemplateHeader = fmt.Sprintf("%s\t%s\t%s", "Name", "Description", "Version") // TODO: add more fields

func getListClusterTemplateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clustertemplates [flags]",
		Aliases: []string{"clustertemplate", "template"},
		Short:   "List all cluster templates",
		RunE:    runListClusterTemplatesCommand,
	}

	cmd.Flags().StringP("namespace", "N", "", "Namespace of the cluster template")

	return cmd
}

func runListClusterTemplatesCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, clusterTemplateClient, projectName, err := getClusterServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := clusterTemplateClient.GetV2ProjectsProjectNameTemplatesWithResponse(ctx, projectName, &coapi.GetV2ProjectsProjectNameTemplatesParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		clusterTemplateHeader, fmt.Sprintf("error getting clusterTemplates")); !proceed {
		return err
	}

	// TODO: implement print
	// printClusterTemplates

	return writer.Flush()
}
