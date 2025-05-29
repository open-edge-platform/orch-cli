// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-edge-platform/cli/pkg/auth"
	coapi "github.com/open-edge-platform/cli/pkg/rest/cluster"
)

const createClusterExamples = `# Create a cluster with the name "my-cluster" on the given nodes using the default template
orch-cli create cluster cli-cluster --project some-project --nodes d7911144-3010-11f0-a1c2-370d26b04195:all

# Create a cluster with the name "my-cluster" using the specified template on the given nodes and with the provided label
orch-cli create cluster cli-cluster --project some-project --nodes d7911144-3010-11f0-a1c2-370d26b04195:all --labels sample-label=samplevalue --template sometemplate-v1.0.0

# Create a cluster with the name "my-cluster" on the given nodes using the default template and with the provided multiple labels
orch-cli create cluster cli-cluster --project some-project --nodes d7911144-3010-11f0-a1c2-370d26b04195:all --labels sample-label=samplevalue,another-label=another-value`

func getCreateClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster <name> [flags]",
		Short:   "Create a cluster",
		Example: createClusterExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runCreateClusterCommand,
	}
	cmd.Flags().String("template", "", "Cluster template to use")
	cmd.Flags().StringSlice("nodes", []string{}, "Mandatory list of nodes in the format <id>:<role>")
	cmd.Flags().StringToString("labels", map[string]string{}, "Labels in the format key=value")
	_ = cmd.MarkFlagRequired("nodes")
	return cmd
}

func getListClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "List clusters",
		Example: "orch-cli list cluster --project some-project",
		RunE:    runListClusterCommand,
	}
	cmd.Flags().String("project", "", "Project name to filter clusters")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func getDeleteClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster <name> [flags]",
		Short:   "Delete a cluster",
		Example: "orch-cli delete cluster cli-cluster --project some-project",
		Args:    cobra.ExactArgs(1),
		RunE:    runDeleteClusterCommand,
	}
	cmd.Flags().Bool("force", false, "Force delete the cluster without waiting for host cleanup")
	cmd.Flags().Bool("verbose", false, "Enable verbose output")
	cmd.Flags().String("project", "", "Project name to filter clusters")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func runCreateClusterCommand(cmd *cobra.Command, args []string) error {
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return processError(err)
	}
	ctx, clusterClient, projectName, err := getClusterServiceContext(cmd)
	if err != nil {
		return err
	}

	clusterName := args[0]

	request := coapi.PostV2ProjectsProjectNameClustersJSONRequestBody{
		Name: &clusterName,
	}

	nodesFlag, err := cmd.Flags().GetStringSlice("nodes")
	if err != nil {
		return processError(err)
	}

	nodes := []coapi.NodeSpec{}
	for _, node := range nodesFlag {
		parts := strings.Split(node, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid node format: %s, expected <id>:<role>", node)
		}
		nodes = append(nodes, coapi.NodeSpec{
			Id:   parts[0],
			Role: coapi.NodeSpecRole(parts[1]),
		})
	}

	request.Nodes = nodes

	template, err := cmd.Flags().GetString("template")
	if err != nil {
		return processError(err)
	}

	if template != "" {
		request.Template = &template
	}

	labels, err := cmd.Flags().GetStringToString("labels")
	if err != nil {
		return processError(err)
	}
	request.Labels = &labels

	if verbose {
		fmt.Printf("Creating cluster with the following details:\n")
		fmt.Printf("- Name: %s\n", *request.Name)
		if template != "" {
			fmt.Printf("- Template: %s\n", template)
		}
		if len(nodes) > 0 {
			fmt.Printf("- Nodes: %+v\n", nodes)
		}
		if request.Labels != nil && len(*request.Labels) > 0 {
			fmt.Printf("- Labels: %v\n", *request.Labels)
		}
	}

	resp, err := clusterClient.PostV2ProjectsProjectNameClustersWithResponse(ctx, projectName, request, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if verbose {
		if resp.JSON201 != nil {
			fmt.Printf("Cluster created successfully: %+v\n", *resp.JSON201)
		} else if resp.Body != nil {
			fmt.Printf("Response body: %s\n", string(resp.Body))
		}
	}

	if resp.JSON201 != nil {
		fmt.Printf("Cluster '%s' created successfully.\n", clusterName)
		return nil
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error creating cluster %s", clusterName))
}

func runListClusterCommand(cmd *cobra.Command, _ []string) error {
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return processError(err)
	}
	ctx, clusterClient, projectName, err := getClusterServiceContext(cmd)
	if err != nil {
		return err
	}

	var clusters []coapi.ClusterInfo
	pageSize := 50
	offset := 0

	for {
		params := coapi.GetV2ProjectsProjectNameClustersParams{
			PageSize: &pageSize,
			Offset:   &offset,
		}
		if verbose {
			fmt.Printf("Fetching clusters for project '%s' with page size %d and offset %d\n", projectName, pageSize, offset)
		}

		resp, err := clusterClient.GetV2ProjectsProjectNameClustersWithResponse(ctx, projectName, &params, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("unexpected response: %s", resp.HTTPResponse.Status)
		}
		if verbose {
			fmt.Printf("Received %d clusters from the server out of total clusters: %d\n", len(*resp.JSON200.Clusters), resp.JSON200.TotalElements)
		}

		clusters = append(clusters, *resp.JSON200.Clusters...)
		if len(*resp.JSON200.Clusters) < pageSize {
			break
		}
		offset += pageSize
	}

	fmt.Printf("Total clusters found: %d\n", len(clusters))
	fmt.Printf("Clusters in project '%s':\n", projectName)
	for i, cluster := range clusters {
		fmt.Printf("%v. %s (%s)\n", i, *cluster.Name, *cluster.ProviderStatus.Message)
	}
	return nil
}

func runDeleteClusterCommand(cmd *cobra.Command, args []string) error {
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return processError(err)
	}
	//force, err := cmd.Flags().GetBool("force")

	ctx, clusterClient, projectName, err := getClusterServiceContext(cmd)
	if err != nil {
		return err
	}

	clusterName := args[0]

	if verbose {
		fmt.Printf("Deleting cluster '%s' in project '%s'\n", clusterName, projectName)
	}

	resp, err := clusterClient.DeleteV2ProjectsProjectNameClustersNameWithResponse(ctx, projectName, clusterName, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	err = checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting cluster %s", clusterName))
	if err != nil {
		fmt.Printf("Failed to delete cluster '%s': %v\n", clusterName, err)
		return err
	}
	fmt.Printf("Cluster '%s' deleted successfully.\n", clusterName)
	return nil
}
