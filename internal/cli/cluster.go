// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	coapi "github.com/open-edge-platform/cli/pkg/rest/cluster"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
)

const (
	DEFAULT_CLUSTER_FORMAT = "table{{.Name}}\t{{.KubernetesVersion}}\t{{statusMessage .LifecyclePhase}}\t{{statusMessage .ProviderStatus}}\t{{statusMessage .ControlPlaneReady}}\t{{statusMessage .InfrastructureReady}}\t{{statusMessage .NodeHealth}}\t{{nodeCount .NodeQuantity}}"
	// List verbose template (uses ClusterInfo with NodeQuantity)
	DEFAULT_CLUSTER_LIST_INSPECT_FORMAT = `Name: {{.Name}}
Kubernetes Version: {{none .KubernetesVersion}}
Node Count: {{nodeCount .NodeQuantity}}
Status:
  Lifecycle Phase: {{statusMessage .LifecyclePhase}}
  Provider Status: {{statusMessage .ProviderStatus}}
  Control Plane Ready: {{statusMessage .ControlPlaneReady}}
  Infrastructure Ready: {{statusMessage .InfrastructureReady}}
  Node Health: {{statusMessage .NodeHealth}}{{if .Labels}}
Labels:{{range $key, $value := deref .Labels}}
  {{$key}}: {{$value}}{{end}}{{else}}
Labels: <none>{{end}}
`
	// Get verbose template (uses ClusterDetailInfo with Template and Nodes)
	DEFAULT_CLUSTER_INSPECT_FORMAT = `Name: {{.Name}}
Kubernetes Version: {{none .KubernetesVersion}}
Template: {{none .Template}}{{if .Nodes}}
Nodes:{{range .Nodes}}
  - ID: {{none .Id}}, Role: {{none .Role}}{{end}}{{else}}
Nodes: <none>{{end}}
Status:
  Lifecycle Phase: {{statusMessage .LifecyclePhase}}
  Provider Status: {{statusMessage .ProviderStatus}}
  Control Plane Ready: {{statusMessage .ControlPlaneReady}}
  Infrastructure Ready: {{statusMessage .InfrastructureReady}}
  Node Health: {{statusMessage .NodeHealth}}{{if .Labels}}
Labels:{{range $key, $value := deref .Labels}}
  {{$key}}: {{$value}}{{end}}{{else}}
Labels: <none>{{end}}
`
	CLUSTER_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_CLUSTER_OUTPUT_TEMPLATE"
)

const createClusterExamples = `
# Create a cluster with the name "my-cluster" on the given nodes using the default template and host resource ID
orch-cli create cluster cli-cluster --project some-project --nodes host-abcd1234:all

# Create a cluster with the name "my-cluster" on the given nodes using the default template and host UUID
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
		Aliases: clusterAliases,
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
		Aliases: clusterAliases,
		RunE:    runGetClusterCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getListClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "List clusters",
		Example: "orch-cli list cluster --project some-project",
		Aliases: clusterAliases,
		RunE:    runListClusterCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "cluster")
	addStandardListOutputFlags(cmd)
	cmd.Flags().Bool("not-ready", false, "Show only clusters that are not ready")
	return cmd
}

func getDeleteClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster <name> [flags]",
		Short:   "Delete a cluster",
		Example: "orch-cli delete cluster cli-cluster",
		Args:    cobra.ExactArgs(1),
		Aliases: clusterAliases,
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
	ctx, clusterClient, projectName, err := ClusterFactory(cmd)
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
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error creating cluster %s", clusterName))
}

func runGetClusterCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, clusterClient, projectName, err := ClusterFactory(cmd)
	if err != nil {
		return err
	}

	clusterName := args[0]

	cluster, err := getClusterDetails(ctx, clusterClient, projectName, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster details: %w", err)
	}

	fmt.Fprintf(writer, "Project: %s\n", projectName)

	// Convert ClusterDetailInfo to ClusterInfo for consistent output
	clusterInfo := coapi.ClusterInfo{
		Name:                cluster.Name,
		KubernetesVersion:   cluster.KubernetesVersion,
		LifecyclePhase:      cluster.LifecyclePhase,
		ProviderStatus:      cluster.ProviderStatus,
		ControlPlaneReady:   cluster.ControlPlaneReady,
		InfrastructureReady: cluster.InfrastructureReady,
		NodeHealth:          cluster.NodeHealth,
		Labels:              cluster.Labels,
	}
	if cluster.Nodes != nil {
		clusterInfo.NodeQuantity = new(int)
		*clusterInfo.NodeQuantity = len(*cluster.Nodes)
	}

	// For verbose output, we need the detail info with nodes
	if verbose {
		// Use the detail template directly
		result := CommandResult{
			Format:    format.Format(DEFAULT_CLUSTER_INSPECT_FORMAT),
			Filter:    "",
			OrderBy:   "",
			OutputAs:  OUTPUT_TABLE,
			NameLimit: -1,
			Data:      cluster,
		}
		GenerateOutput(writer, &result)
	} else {
		// For non-verbose, show a single cluster in table format
		clusters := []coapi.ClusterInfo{clusterInfo}
		var emptyFilter string
		if err := printClusters(cmd, writer, &clusters, nil, &emptyFilter, false); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func getValidatedClusterOrderBy(
	ctx context.Context,
	cmd *cobra.Command,
	clusterClient coapi.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	// For table format (default), use client-side sorting which supports any field in the model
	if outputType == "table" {
		return normalizeOrderByForClientSorting(raw, coapi.ClusterInfo{})
	}

	// For JSON/YAML, use API ordering (only API-supported fields)
	return normalizeOrderByWithAPIProbe(raw, "clusters", coapi.ClusterInfo{}, func(orderBy string) (bool, error) {
		pageSize := 1
		offset := 0
		// Validate ordering in isolation. Reusing the caller's --filter here can turn
		// filter errors into misleading "invalid --order-by field" errors.
		resp, err := clusterClient.GetV2ProjectsProjectNameClustersWithResponse(ctx, projectName,
			&coapi.GetV2ProjectsProjectNameClustersParams{
				OrderBy:  &orderBy,
				Filter:   nil,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating cluster order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func getValidatedClusterFilter(
	ctx context.Context,
	cmd *cobra.Command,
	clusterClient coapi.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return normalizeFilterWithAPIProbe(raw, "clusters", coapi.ClusterInfo{}, func(filter string) (bool, error) {
		pageSize := 1
		offset := 0
		resp, err := clusterClient.GetV2ProjectsProjectNameClustersWithResponse(ctx, projectName,
			&coapi.GetV2ProjectsProjectNameClustersParams{
				OrderBy:  nil,
				Filter:   &filter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating cluster filter"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func printClusters(cmd *cobra.Command, writer io.Writer, clusterList *[]coapi.ClusterInfo, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getClusterOutputFormat(cmd, verbose, true)
	if err != nil {
		return err
	}

	sortSpec := ""
	if outputType == "table" && orderBy != nil {
		sortSpec = *orderBy
	}

	filterSpec := ""
	if outputType == "table" && outputFilter != nil && *outputFilter != "" {
		filterSpec = *outputFilter
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OrderBy:   sortSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      *clusterList,
	}

	GenerateOutput(writer, &result)
	return nil
}

func getClusterOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	if verbose {
		if forList {
			return DEFAULT_CLUSTER_LIST_INSPECT_FORMAT, nil
		}
		return DEFAULT_CLUSTER_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_CLUSTER_FORMAT, CLUSTER_OUTPUT_TEMPLATE_ENVVAR)
}

func runListClusterCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, clusterClient, projectName, err := ClusterFactory(cmd)
	if err != nil {
		return err
	}

	validatedOrderBy, err := getValidatedClusterOrderBy(ctx, cmd, clusterClient, projectName)
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		apiOrderBy = nil
	}

	validatedFilter, err := getValidatedClusterFilter(ctx, cmd, clusterClient, projectName)
	if err != nil {
		return err
	}

	pageSize32, offset32, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	// Convert int32 to int for cluster API
	pageSize := int(pageSize32)
	offset := int(offset32)

	// Cluster API requires pageSize > 0, use default if not specified
	if pageSize <= 0 {
		pageSize = 100
	}

	notReady, _ := cmd.Flags().GetBool("not-ready")

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		resp, err := clusterClient.GetV2ProjectsProjectNameClustersWithResponse(ctx, projectName,
			&coapi.GetV2ProjectsProjectNameClustersParams{
				OrderBy:  apiOrderBy,
				Filter:   validatedFilter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing clusters"); !proceed {
			return err
		}
		if resp.JSON200 == nil || resp.JSON200.Clusters == nil {
			return fmt.Errorf("error listing clusters: unexpected response format")
		}
		clusters := *resp.JSON200.Clusters
		if notReady {
			clusters = filterNotReadyClusters(clusters)
		}
		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printClusters(cmd, writer, &clusters, validatedOrderBy, &outputFilter, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	allClusters := make([]coapi.ClusterInfo, 0)

	resp, err := clusterClient.GetV2ProjectsProjectNameClustersWithResponse(ctx, projectName,
		&coapi.GetV2ProjectsProjectNameClustersParams{
			OrderBy:  apiOrderBy,
			Filter:   validatedFilter,
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		"error listing clusters"); !proceed {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Clusters == nil {
		return fmt.Errorf("error listing clusters: unexpected response format")
	}

	allClusters = append(allClusters, *resp.JSON200.Clusters...)
	totalElements := int(resp.JSON200.TotalElements)

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = len(*resp.JSON200.Clusters)
	}

	for len(allClusters) < totalElements {
		if pageSize <= 0 {
			break
		}
		offset += pageSize
		resp, err := clusterClient.GetV2ProjectsProjectNameClustersWithResponse(ctx, projectName,
			&coapi.GetV2ProjectsProjectNameClustersParams{
				OrderBy:  apiOrderBy,
				Filter:   validatedFilter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
			"error listing clusters"); !proceed {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.Clusters == nil {
			return fmt.Errorf("error listing clusters: unexpected response format")
		}

		if len(*resp.JSON200.Clusters) == 0 {
			break
		}
		allClusters = append(allClusters, *resp.JSON200.Clusters...)
	}

	if notReady {
		allClusters = filterNotReadyClusters(allClusters)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printClusters(cmd, writer, &allClusters, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func filterNotReadyClusters(clusters []coapi.ClusterInfo) []coapi.ClusterInfo {
	filtered := make([]coapi.ClusterInfo, 0)
	for _, cluster := range clusters {
		if !clusterReady(cluster) {
			filtered = append(filtered, cluster)
		}
	}
	return filtered
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

	ctx, clusterClient, projectName, err := ClusterFactory(cmd)
	if err != nil {
		return err
	}

	clusterName := args[0]

	fmt.Printf("Deleting cluster '%s' in project '%s'\n", clusterName, projectName)
	if force {
		ctx, hostClient, projectName, err := InfraFactory(cmd)
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

func getClusterDetails(ctx context.Context, clusterClient coapi.ClientWithResponsesInterface, projectName, clusterName string) (res coapi.ClusterDetailInfo, err error) {
	resp, err := clusterClient.GetV2ProjectsProjectNameClustersNameWithResponse(ctx, projectName, clusterName, auth.AddAuthHeader)
	if err != nil {
		return res, processError(err)
	}
	if resp.JSON200 == nil {
		return res, fmt.Errorf("cluster %s not found in project %s", clusterName, projectName)
	}
	return *resp.JSON200, nil
}

func softDeleteCluster(ctx context.Context, clusterClient coapi.ClientWithResponsesInterface, projectName, clusterName string) error {
	resp, err := clusterClient.DeleteV2ProjectsProjectNameClustersNameWithResponse(ctx, projectName, clusterName, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if resp.HTTPResponse.StatusCode != 204 {
		return fmt.Errorf("failed to delete cluster %s: %s", clusterName, resp.HTTPResponse.Status)
	}
	return nil
}

func forceDeleteCluster(ctx context.Context, hostClient infra.ClientWithResponsesInterface, clusterClient coapi.ClientWithResponsesInterface, projectName, clusterName string) error {
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
		if resp.HTTPResponse.StatusCode != 204 && resp.HTTPResponse.StatusCode != 200 {
			return fmt.Errorf("failed to delete node %s from cluster %s: %s", uuid, clusterName, resp.HTTPResponse.Status)
		}
		fmt.Printf("Node %s deleted successfully from cluster %s\n", uuid, clusterName)

	}

	return nil
}

func getHostUUID(ctx context.Context, hostClient infra.ClientWithResponsesInterface, projectName, hostID string) (string, error) {
	resp, err := hostClient.HostServiceGetHostWithResponse(ctx, projectName, hostID, auth.AddAuthHeader)
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
	return *uuid, nil
}
