// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-edge-platform/cli/pkg/auth"
	coapi "github.com/open-edge-platform/cli/pkg/rest/cluster"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
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

func getGetClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster <name>",
		Short:   "Get details of a cluster",
		Example: "orch-cli get cluster cli-cluster",
		Args:    cobra.ExactArgs(1),
		RunE:    runGetClusterCommand,
	}
	return cmd
}

func getListClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "List clusters",
		Example: "orch-cli list cluster",
		RunE:    runListClusterCommand,
	}
	cmd.Flags().Bool("not-ready", false, "Show only clusters that are not ready")
	return cmd
}

func getDeleteClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster <name> [flags]",
		Short:   "Delete a cluster",
		Example: "orch-cli delete cluster cli-cluster",
		Args:    cobra.ExactArgs(1),
		RunE:    runDeleteClusterCommand,
	}
	cmd.Flags().Bool("force", false, "Force delete the cluster without waiting for the host cleanup")
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

func runGetClusterCommand(cmd *cobra.Command, args []string) error {
	ctx, clusterClient, projectName, err := getClusterServiceContext(cmd)
	if err != nil {
		return err
	}

	clusterName := args[0]

	cluster, err := getClusterDetails(ctx, clusterClient, projectName, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster details: %w", err)
	}

	fmt.Printf("Project: %s\n", projectName)
	fmt.Printf("Name: %s\n", *cluster.Name)
	fmt.Printf("Kubernetes Version: %s\n", *cluster.KubernetesVersion)
	fmt.Printf("Template: %s\n", *cluster.Template)
	fmt.Printf("Nodes:\n")
	if cluster.Nodes != nil {
		for _, node := range *cluster.Nodes {
			id := ""
			if node.Id != nil {
				id = *node.Id
			}
			role := ""
			if node.Role != nil {
				role = string(*node.Role)
			}
			fmt.Printf("- ID: %s, Role: %s\n", id, role)
		}
	}
	statusUnknown := "<unknown>"

	fmt.Printf("Status:\n")
	lifecyclePhase := statusUnknown
	if cluster.LifecyclePhase != nil && cluster.LifecyclePhase.Message != nil {
		lifecyclePhase = *cluster.LifecyclePhase.Message
	}
	fmt.Printf("- LifecyclePhase: %s\n", lifecyclePhase)

	providerStatus := statusUnknown
	if cluster.ProviderStatus != nil && cluster.ProviderStatus.Message != nil {
		providerStatus = *cluster.ProviderStatus.Message
	}
	fmt.Printf("- Provider: %s\n", providerStatus)

	controlPlaneReady := statusUnknown
	if cluster.ControlPlaneReady != nil && cluster.ControlPlaneReady.Message != nil {
		controlPlaneReady = *cluster.ControlPlaneReady.Message
	}
	fmt.Printf("- ControlPlaneReady: %s\n", controlPlaneReady)

	infrastructureReady := statusUnknown
	if cluster.InfrastructureReady != nil && cluster.InfrastructureReady.Message != nil {
		infrastructureReady = *cluster.InfrastructureReady.Message
	}
	fmt.Printf("- InfrastructureReady: %s\n", infrastructureReady)

	nodeHealth := statusUnknown
	if cluster.NodeHealth != nil && cluster.NodeHealth.Message != nil {
		nodeHealth = *cluster.NodeHealth.Message
	}
	fmt.Printf("- NodeHealth: %s\n", nodeHealth)

	if cluster.Labels != nil {
		fmt.Printf("Labels:\n")
		for key, value := range *cluster.Labels {
			fmt.Printf("- %s: %s\n", key, value)
		}
	} else {
		fmt.Println("Labels: None")
	}
	return nil
}

func runListClusterCommand(cmd *cobra.Command, _ []string) error {
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return processError(err)
	}
	notReady, err := cmd.Flags().GetBool("not-ready")
	if err != nil {
		return processError(err)
	}

	ctx, clusterClient, projectName, err := getClusterServiceContext(cmd)
	if err != nil {
		return err
	}

	var clusters []coapi.ClusterInfo
	pageSize := 100
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

	fmt.Printf("Found %d clusters in project '%s'\n", len(clusters), projectName)
	filteredClusters := 0
	result := "Clusters:\n"
	for _, cluster := range clusters {
		if notReady && clusterReady(cluster) {
			continue
		}
		filteredClusters++
		result += fmt.Sprintf("- %s (%s)\n", *cluster.Name, statusMessage(cluster))
	}
	if notReady {
		fmt.Printf("Found %d clusters that are not ready in project '%s'\n", filteredClusters, projectName)
	}
	fmt.Println(result)
	return nil
}

func clusterReady(cluster coapi.ClusterInfo) bool {
	if cluster.LifecyclePhase == nil || *cluster.LifecyclePhase.Indicator != coapi.STATUSINDICATIONIDLE {
		return false
	}
	if cluster.ProviderStatus == nil || *cluster.ProviderStatus.Indicator != coapi.STATUSINDICATIONIDLE {
		return false
	}
	if cluster.ControlPlaneReady == nil || *cluster.ControlPlaneReady.Indicator != coapi.STATUSINDICATIONIDLE {
		return false
	}
	if cluster.InfrastructureReady == nil || *cluster.InfrastructureReady.Indicator != coapi.STATUSINDICATIONIDLE {
		return false
	}
	if cluster.NodeHealth == nil || *cluster.NodeHealth.Indicator != coapi.STATUSINDICATIONIDLE {
		return false
	}

	return true
}

func statusMessage(cluster coapi.ClusterInfo) string {
	if cluster.ProviderStatus == nil || *cluster.ProviderStatus.Indicator != coapi.STATUSINDICATIONIDLE {
		return *cluster.ProviderStatus.Message
	}
	if cluster.ControlPlaneReady == nil || *cluster.ControlPlaneReady.Indicator != coapi.STATUSINDICATIONIDLE {
		return *cluster.ControlPlaneReady.Message
	}
	if cluster.InfrastructureReady == nil || *cluster.InfrastructureReady.Indicator != coapi.STATUSINDICATIONIDLE {
		return *cluster.InfrastructureReady.Message
	}
	if cluster.NodeHealth == nil || *cluster.NodeHealth.Indicator != coapi.STATUSINDICATIONIDLE {
		return *cluster.NodeHealth.Message
	}
	if cluster.LifecyclePhase == nil || *cluster.LifecyclePhase.Indicator != coapi.STATUSINDICATIONIDLE {
		return *cluster.LifecyclePhase.Message
	}
	return "active"
}

func runDeleteClusterCommand(cmd *cobra.Command, args []string) error {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return processError(err)
	}

	ctx, clusterClient, projectName, err := getClusterServiceContext(cmd)
	if err != nil {
		return err
	}

	clusterName := args[0]

	fmt.Printf("Deleting cluster '%s' in project '%s'\n", clusterName, projectName)
	if force {
		ctx, hostClient, projectName, err := getInfraServiceContext(cmd)
		if err != nil {
			return fmt.Errorf("failed to get infra service context: %w", err)
		}
		err = forceDeleteCluster(ctx, hostClient, clusterClient, projectName, clusterName)
		if err != nil {
			return fmt.Errorf("failed to force delete cluster '%s': %w", clusterName, err)
		}
	} else {
		err = softDeleteCluster(ctx, clusterClient, projectName, clusterName)
		if err != nil {
			return fmt.Errorf("failed to soft delete cluster '%s': %w", clusterName, err)
		}
	}
	fmt.Printf("Cluster '%s' deletion initiated successfully.\n", clusterName)
	return nil
}

func getClusterDetails(ctx context.Context, clusterClient *coapi.ClientWithResponses, projectName, clusterName string) (res coapi.ClusterDetailInfo, err error) {
	resp, err := clusterClient.GetV2ProjectsProjectNameClustersNameWithResponse(ctx, projectName, clusterName, auth.AddAuthHeader)
	if err != nil {
		return res, processError(err)
	}
	if resp.JSON200 == nil {
		return res, fmt.Errorf("cluster %s not found in project %s", clusterName, projectName)
	}
	return *resp.JSON200, nil
}

func softDeleteCluster(ctx context.Context, clusterClient *coapi.ClientWithResponses, projectName, clusterName string) error {
	resp, err := clusterClient.DeleteV2ProjectsProjectNameClustersNameWithResponse(ctx, projectName, clusterName, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if resp.HTTPResponse.StatusCode != 204 {
		return fmt.Errorf("failed to delete cluster %s: %s", clusterName, resp.HTTPResponse.Status)
	}
	return nil
}

func forceDeleteCluster(ctx context.Context, hostClient *infra.ClientWithResponses, clusterClient *coapi.ClientWithResponses, projectName, clusterName string) error {
	cluster, err := getClusterDetails(ctx, clusterClient, projectName, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster details for force delete: %w", err)
	}

	for _, node := range *cluster.Nodes {
		if node.Id == nil || *node.Id == "" {
			return fmt.Errorf("node ID is missing for node in cluster %s", clusterName)
		}
		hostID := *node.Id
		uuid, err := getHostUUID(ctx, hostClient, projectName, hostID)
		if err != nil {
			return fmt.Errorf("failed to get UUID for host %s: %w", *node.Id, err)
		}
		fmt.Printf("Force deleting node %s from cluster %s\n", uuid, clusterName)
		force := true
		params := coapi.DeleteV2ProjectsProjectNameClustersNameNodesNodeIdParams{
			Force: &force,
		}
		resp, err := clusterClient.DeleteV2ProjectsProjectNameClustersNameNodesNodeIdWithResponse(ctx, projectName, clusterName, uuid, &params, auth.AddAuthHeader)
		if err != nil {
			return fmt.Errorf("failed to delete node %s from cluster %s: %w", uuid, clusterName, err)
		}
		if resp.HTTPResponse.StatusCode != 204 {
			return fmt.Errorf("failed to delete node %s from cluster %s: %s", uuid, clusterName, resp.HTTPResponse.Status)
		}
		fmt.Printf("Node %s deleted successfully from cluster %s\n", uuid, clusterName)

	}

	return nil
}

func getHostUUID(ctx context.Context, hostClient *infra.ClientWithResponses, projectName, hostID string) (string, error) {
	resp, err := hostClient.GetV1ProjectsProjectNameComputeHostsHostIDWithResponse(ctx, projectName, hostID, auth.AddAuthHeader)
	if err != nil {
		return "", processError(err)
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("host %s not found in project %s", hostID, projectName)
	}

	host := resp.JSON200
	uuid := host.Uuid
	if uuid == nil {
		return "", fmt.Errorf("host %s does not have a UUID", hostID)
	}
	return uuid.String(), nil
}
