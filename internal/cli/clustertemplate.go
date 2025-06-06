// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"

	coapi "github.com/open-edge-platform/cli/pkg/rest/cluster"
)

var clusterTemplateHeader = fmt.Sprintf("%s\t%s\t%s", "Name", "Description", "Version")

// toJSON is a helper function to format a struct or map into a nicely formatted JSON string
func toJSON(s interface{}) string {
	if s == nil {
		return "nil"
	}

	formatted, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Sprintf("error formatting clusterConfiguration: %v", err)
	}

	return string(formatted)
}

func printClusterTemplates(writer io.Writer, clusterTemplates *[]coapi.TemplateInfo, verbose bool) {
	for _, ct := range *clusterTemplates {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", ct.Name, *ct.Description, ct.Version)
		} else {
			_, _ = fmt.Fprintf(writer, "Name: %s\n", ct.Name)
			_, _ = fmt.Fprintf(writer, "Description: %s\n", *ct.Description)
			_, _ = fmt.Fprintf(writer, "Version: %s\n", ct.Version)
			_, _ = fmt.Fprintf(writer, "KubernetesVersion: %s\n", ct.KubernetesVersion)
			_, _ = fmt.Fprintf(writer, "Controlplaneprovidertype: %s\n", *ct.Controlplaneprovidertype)
			_, _ = fmt.Fprintf(writer, "Infraprovidertype: %s\n", *ct.Infraprovidertype)
			_, _ = fmt.Fprintf(writer, "ClusterLabels: %v\n", toJSON(ct.ClusterLabels))
			_, _ = fmt.Fprintf(writer, "ClusterNetwork: %v\n", toJSON(ct.ClusterNetwork))
			_, _ = fmt.Fprintf(writer, "Clusterconfiguration: %v\n", toJSON(ct.Clusterconfiguration))
		}
	}
}

func getListClusterTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clustertemplates [flags]",
		Aliases: []string{"clustertemplate", "template"},
		Short:   "List all cluster templates",
		Example: "orch-cli list clustertemplates --project some-project",
		RunE:    runListClusterTemplatesCommand,
	}
	return cmd
}

func runListClusterTemplatesCommand(cmd *cobra.Command, _ []string) error {
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
		clusterTemplateHeader, "error getting clusterTemplates"); !proceed {
		return err
	}

	printClusterTemplates(writer, resp.JSON200.TemplateInfoList, verbose)

	return writer.Flush()
}
