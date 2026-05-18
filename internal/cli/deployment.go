// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/open-edge-platform/cli/internal/validator"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	depapi "github.com/open-edge-platform/cli/pkg/rest/deployment"
	"github.com/spf13/cobra"
)

// Context key types for passing expanded values
type contextKey string

const (
	expandedLabelsKey                 contextKey = "expandedLabels"
	expandedClusterIDsKey             contextKey = "expandedClusterIDs"
	DEFAULT_DEPLOYMENT_FORMAT                    = "table{{str .DeployId}}\t{{str .Name}}\t{{str .DisplayName}}\t{{str .ProfileName}}\t{{.Status.State}}"
	DEFAULT_DEPLOYMENT_INSPECT_FORMAT            = `Deployment ID: {{str .DeployId}}
Name: {{str .Name}}
Display Name: {{str .DisplayName}}
Profile: {{str .ProfileName}}
State: {{.Status.State}}
Create Time: {{.CreateTime}}
`
	DEPLOYMENT_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_DEPLOYMENT_OUTPUT_TEMPLATE"
)

func getCreateDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <package-name> <version> [flags]",
		Short:   "Create a deployment",
		Example: "orch-cli create deployment my-package 1.0.0 --project sample-project --display-name my-deployment --profile sample-profile --application-label <label>=<label-value>",
		Args:    cobra.ExactArgs(2),
		Aliases: deploymentAliases,
		RunE:    runCreateDeploymentCommand,
	}
	cmd.Flags().String("display-name", "", "deployment display name")
	cmd.Flags().String("profile", "", "deployment profile to use")
	cmd.Flags().StringToString("application-namespace", map[string]string{}, "application target namespaces in format '<app>=<namespace>'")
	cmd.Flags().StringToString("application-set", map[string]string{}, "application set value overrides in form of '<app>.<prop>=<prop-value>'")
	cmd.Flags().StringToString("application-label", map[string]string{}, "labels to deploy ALL applications in the package to matching clusters in format '<label>=<value>'")
	cmd.Flags().String("application-cluster-id", "", "cluster ID to deploy ALL applications in the package to")
	return cmd
}

func getListDeploymentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployments [flags]",
		Aliases: deploymentAliases,
		Short:   "List all deployments",
		Example: "orch-cli list deployments --project some-project",
		RunE:    runListDeploymentsCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "deployment")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getGetDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <deployment-id|display-name> [flags]",
		Aliases: deploymentAliases,
		Short:   "Get a deployment",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli get deployment 9d652bb4-2412-4566-89b5-614f22e2a837 --project some-project\norch-cli get deployment \"Test Headlamp Dashboard\" --project some-project",
		RunE:    runGetDeploymentCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getSetDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <deployment-id|display-name> [flags]",
		Short:   "Update a deployment",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli set deployment 9d652bb4-2412-4566-89b5-614f22e2a837 --project some-project --profile sample-profile\norch-cli set deployment \"Test Headlamp Dashboard\" --project some-project --profile sample-profile",
		Aliases: deploymentAliases,
		RunE:    runSetDeploymentCommand,
	}
	cmd.Flags().String("name", "", "deployment name")
	cmd.Flags().String("package-name", "", "deployment package name")
	cmd.Flags().String("package-version", "", "deployment package version")
	cmd.Flags().String("profile", "", "deployment profile to use")
	cmd.Flags().StringToString("application-namespace", map[string]string{}, "application target namespaces in format '<app>=<namespace>'")
	cmd.Flags().StringToString("application-set", map[string]string{}, "application set value overrides in form of '<app>.<prop>=<prop-value>'")
	cmd.Flags().StringToString("application-label", map[string]string{}, "labels to deploy ALL applications in the package to matching clusters in format '<label>=<value>'")
	cmd.Flags().String("application-cluster-id", "", "cluster ID to deploy ALL applications in the package to")
	return cmd
}

func getUpgradeDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <deployment-id|display-name> [flags]",
		Short:   "Upgrade a deployment to a new package version",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli upgrade deployment 9d652bb4-2412-4566-89b5-614f22e2a837 --package-version 1.1.0\norch-cli upgrade deployment \"Test Headlamp Dashboard\" --package-version 1.1.0",
		Aliases: deploymentAliases,
		RunE:    runUpgradeDeploymentCommand,
	}
	cmd.Flags().String("package-version", "", "new deployment package version to upgrade to")
	_ = cmd.MarkFlagRequired("package-version")
	return cmd
}

func getDeleteDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <deployment-id|display-name> [flags]",
		Short:   "Delete a deployment",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli delete deployment 9d652bb4-2412-4566-89b5-614f22e2a837 --project some-project\norch-cli delete deployment \"Test Headlamp Dashboard\" --project some-project",
		Aliases: deploymentAliases,
		RunE:    runDeleteDeploymentCommand,
	}
	return cmd
}

func getDeploymentOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_DEPLOYMENT_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_DEPLOYMENT_FORMAT, DEPLOYMENT_OUTPUT_TEMPLATE_ENVVAR)
}

func printDeployments(cmd *cobra.Command, writer *tabwriter.Writer, deployments *[]depapi.DeploymentV1Deployment, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getDeploymentOutputFormat(cmd, verbose)
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
		Data:      *deployments,
	}

	GenerateOutput(writer, &result)
	return nil
}

func getValidatedDeploymentOrderBy(
	ctx context.Context,
	cmd *cobra.Command,
	deploymentClient depapi.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	// For table format (default), use client-side sorting which supports any field in the model
	if outputType == "table" {
		return normalizeOrderByForClientSorting(raw, depapi.DeploymentV1Deployment{})
	}

	// For JSON/YAML, use API ordering (only API-supported fields)
	return normalizeOrderByWithAPIProbe(raw, "deployments", depapi.DeploymentV1Deployment{}, func(orderBy string) (bool, error) {
		pageSize := int32(1)
		offset := int32(0)
		// Validate ordering in isolation. Reusing the caller's --filter here can turn
		// filter errors into misleading "invalid --order-by field" errors.
		resp, err := deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
			&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{
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
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating deployment order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func runCreateDeploymentCommand(cmd *cobra.Command, args []string) error {
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	appName := args[0]
	appVersion := args[1]

	// Validate version format
	if err := validator.ValidateVersion(appVersion); err != nil {
		return err
	}

	// Get catalog client for fetching deployment package
	_, catalogClient, _, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	// Get valid application names from the deployment package for validation
	validAppNames, err := getValidApplicationNames(ctx, catalogClient, projectName, appName, appVersion)
	if err != nil {
		return err
	}

	// Expand --application-label to apply to ALL applications in the package
	// Format: <label>=<value> applies the same label to all apps
	labels, _ := cmd.Flags().GetStringToString("application-label")
	if len(labels) > 0 {
		expandedLabels := make(map[string]string)
		for appName := range validAppNames {
			for label, value := range labels {
				key := fmt.Sprintf("%s.%s", appName, label)
				expandedLabels[key] = value
			}
		}
		cmd.SetContext(context.WithValue(cmd.Context(), expandedLabelsKey, expandedLabels))
	}

	// Expand --application-cluster-id to apply to ALL applications in the package
	// Format: <cluster-id> applies the same cluster to all apps
	clusterID, _ := cmd.Flags().GetString("application-cluster-id")
	if clusterID != "" {
		expandedClusterIDs := make(map[string]string)
		for appName := range validAppNames {
			expandedClusterIDs[appName] = clusterID
		}
		cmd.SetContext(context.WithValue(cmd.Context(), expandedClusterIDsKey, expandedClusterIDs))
	}

	overrideValues, err := getOverrideValues(cmd)
	if err != nil {
		return err
	}

	// Validate that application names in overrides exist in the deployment package
	if err := validateApplicationNames(overrideValues, validAppNames, "application-set"); err != nil {
		return err
	}

	targetClusters, deploymentType, err := getTargetClusters(cmd, false) // do not allow empty target clusters
	if err != nil {
		return err
	}

	// Validate that application names in target clusters exist in the deployment package
	if err := validateTargetClustersApplicationNames(targetClusters, validAppNames); err != nil {
		return err
	}

	resp, err := deploymentClient.DeploymentV1DeploymentServiceCreateDeploymentWithResponse(ctx, projectName,
		depapi.DeploymentV1DeploymentServiceCreateDeploymentJSONRequestBody{
			DisplayName:    getFlag(cmd, "display-name"),
			AppName:        appName,
			AppVersion:     appVersion,
			ProfileName:    getFlag(cmd, "profile"),
			OverrideValues: &overrideValues,
			TargetClusters: targetClusters,
			DeploymentType: &deploymentType,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error creating deployment for application %s:%s", appName, appVersion)); err != nil {
		return err
	}
	// Extract deployment ID from response if available
	if resp.JSON200 != nil && resp.JSON200.DeploymentId != "" {
		fmt.Printf("Deployment created successfully (ID: %s)\n", resp.JSON200.DeploymentId)
	} else {
		fmt.Printf("Deployment for '%s:%s' created successfully\n", appName, appVersion)
	}
	return nil
}

func getOverrideValues(cmd *cobra.Command) ([]depapi.DeploymentV1OverrideValues, error) {
	namespaces, _ := cmd.Flags().GetStringToString("application-namespace")
	sets, _ := cmd.Flags().GetStringToString("application-set")
	return getOverrideValuesRaw(namespaces, sets)
}

func getOverrideValuesRaw(namespaces, sets map[string]string) ([]depapi.DeploymentV1OverrideValues, error) {
	overrides := make(map[string]depapi.DeploymentV1OverrideValues, 0)

	// First, accumulate any app target namespace settings in the format "<app-name>=<target-namespace>"
	for app, namespace := range namespaces {
		ns := namespace
		values := make(depapi.GoogleProtobufStruct, 0)
		overrides[app] = depapi.DeploymentV1OverrideValues{AppName: app, TargetNamespace: &ns, Values: &values}
	}

	// Next accumulate any app value overrides in format "<app-name>.<property-name>=<value>"
	for appProp, value := range sets {
		fields := strings.SplitN(appProp, ".", 2)
		if len(fields) < 2 {
			return nil, fmt.Errorf("property %s not in format <app-name>.<property-name>", appProp)
		}
		app := fields[0]
		prop := fields[1]

		override, ok := overrides[app]
		if !ok {
			values := make(depapi.GoogleProtobufStruct, 0)
			override = depapi.DeploymentV1OverrideValues{AppName: app, Values: &values}
		}
		override.Values = addProperty(override.Values, app, prop, value)
		overrides[app] = override
	}

	// Transform overrides map into array
	overrideValues := make([]depapi.DeploymentV1OverrideValues, 0)
	for _, override := range overrides {
		overrideValues = append(overrideValues, override)
	}
	return overrideValues, nil
}

func addProperty(values *depapi.GoogleProtobufStruct, _ string, prop string, value string) *depapi.GoogleProtobufStruct {
	ovs := *values
	segs := strings.SplitN(prop, ".", 2)
	if _, ok := ovs[segs[0]]; !ok {
		gv := depapi.GoogleProtobufValue{}
		ovs[segs[0]] = &gv
	}
	if len(segs) == 1 {
		(*ovs[segs[0]])[segs[0]] = parseValue(value)
	} else {
		addToProtobufValue((*map[string]interface{})(ovs[segs[0]]), segs[1], value)
	}
	return values
}

func addToProtobufValue(m *map[string]interface{}, prop string, value string) {
	ovs := *m
	segs := strings.SplitN(prop, ".", 2)
	if len(segs) == 1 {
		ovs[segs[0]] = parseValue(value)
	} else {
		if _, ok := ovs[segs[0]]; !ok {
			newProps := make(map[string]interface{})
			ovs[segs[0]] = &newProps
		}
		addToProtobufValue(ovs[segs[0]].(*map[string]interface{}), segs[1], value)
	}
}

func parseValue(value string) interface{} {
	if v, err := strconv.ParseBool(value); err == nil {
		return v
	}
	if v, err := strconv.ParseInt(value, 10, 32); err == nil {
		return v
	}
	if v, err := strconv.ParseFloat(value, 64); err == nil {
		return v
	}
	return value
}

func getTargetClusters(cmd *cobra.Command, allowEmpty bool) (*[]depapi.DeploymentV1TargetClusters, string, error) {
	targetClustersByLabel, err := getTargetClustersByLabel(cmd)
	if err != nil {
		return nil, "", err
	}

	targetClustersByID, err := getTargetClustersByID(cmd)
	if err != nil {
		return nil, "", err
	}

	if targetClustersByLabel != nil && len(*targetClustersByLabel) > 0 {
		// ADM does not allow a deployment to be both Automatic and Manual,
		// so both labels and cluster-ids are not allowed at the same time.
		if targetClustersByID != nil && len(*targetClustersByID) > 0 {
			return nil, "", fmt.Errorf("cannot specify both application-label and application-cluster-id flags")
		}
		return targetClustersByLabel, "auto-scaling", nil
	} else if targetClustersByID != nil && len(*targetClustersByID) > 0 {
		return targetClustersByID, "targeted", nil
	}

	if !allowEmpty {
		return nil, "", fmt.Errorf("no target clusters specified, use either --application-label or --application-cluster-id")
	}
	return &[]depapi.DeploymentV1TargetClusters{}, "", nil
}

func getTargetClustersByLabel(cmd *cobra.Command) (*[]depapi.DeploymentV1TargetClusters, error) {
	// Check for expanded labels from context (set during command preprocessing)
	var labels map[string]string
	if expandedLabels := cmd.Context().Value(expandedLabelsKey); expandedLabels != nil {
		labels = expandedLabels.(map[string]string)
	} else {
		// Fall back to flag value (for backward compatibility with old format)
		labels, _ = cmd.Flags().GetStringToString("application-label")
	}

	// Accumulate any app target cluster labels in format "<app-name>.<label-name>=<label-value>"
	targets := make(map[string]depapi.DeploymentV1TargetClusters, 0)
	for appLabel, value := range labels {
		fields := strings.SplitN(appLabel, ".", 2)
		if len(fields) < 2 {
			return nil, fmt.Errorf("label %s not in format <app-name>.<label-name>", appLabel)
		}
		app := fields[0]
		label := fields[1]

		target, alreadyExists := targets[app]

		if !alreadyExists {
			lbls := make(map[string]string, 1)
			target = depapi.DeploymentV1TargetClusters{AppName: &app, Labels: &lbls}
			targets[app] = target
		}
		(*target.Labels)[label] = value
	}

	// Transform targets map into array
	targetClusters := make([]depapi.DeploymentV1TargetClusters, 0, len(targets))
	for _, target := range targets {
		targetClusters = append(targetClusters, target)
	}
	return &targetClusters, nil
}

func getTargetClustersByID(cmd *cobra.Command) (*[]depapi.DeploymentV1TargetClusters, error) {
	// Check for expanded cluster IDs from context (set during command preprocessing)
	var clusterIDs map[string]string
	if expandedIDs := cmd.Context().Value(expandedClusterIDsKey); expandedIDs != nil {
		clusterIDs = expandedIDs.(map[string]string)
	}

	if len(clusterIDs) == 0 {
		return &[]depapi.DeploymentV1TargetClusters{}, nil
	}

	// Transform to target clusters array
	targetClusters := make([]depapi.DeploymentV1TargetClusters, 0, len(clusterIDs))
	for app, clusterID := range clusterIDs {
		appName := app
		cID := clusterID
		target := depapi.DeploymentV1TargetClusters{AppName: &appName, ClusterId: &cID}
		targetClusters = append(targetClusters, target)
	}
	return &targetClusters, nil
}

func runListDeploymentsCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	validatedOrderBy, err := getValidatedDeploymentOrderBy(ctx, cmd, deploymentClient, projectName)
	if err != nil {
		return err
	}

	validatedFilter, err := getValidatedDeploymentFilter(ctx, cmd, deploymentClient, projectName)
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		apiOrderBy = nil
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		resp, err := deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
			&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{
				OrderBy:  apiOrderBy,
				Filter:   validatedFilter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
			"", "error getting deployments"); !proceed {
			return err
		}
		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printDeployments(cmd, writer, &resp.JSON200.Deployments, validatedOrderBy, &outputFilter, verbose); err != nil {
			return err
		}
		return writer.Flush()
	}

	allDeployments := make([]depapi.DeploymentV1Deployment, 0)

	resp, err := deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
		&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{
			OrderBy:  apiOrderBy,
			Filter:   validatedFilter,
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting deployments"); !proceed {
		return err
	}

	allDeployments = append(allDeployments, resp.JSON200.Deployments...)
	totalElements := int(resp.JSON200.TotalElements)

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = int32(len(resp.JSON200.Deployments))
	}

	for len(allDeployments) < totalElements {
		if pageSize <= 0 {
			break
		}

		offset += pageSize
		resp, err = deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
			&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{
				OrderBy:  apiOrderBy,
				Filter:   validatedFilter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
			"", "error getting deployments"); !proceed {
			return err
		}

		if len(resp.JSON200.Deployments) == 0 {
			break
		}
		allDeployments = append(allDeployments, resp.JSON200.Deployments...)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printDeployments(cmd, writer, &allDeployments, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

// deploymentIDPattern matches deployment UUIDs.
var deploymentIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func isDeploymentID(s string) bool {
	return deploymentIDPattern.MatchString(s)
}

// findDeploymentByDisplayName searches a slice of deployments for an exact DisplayName match.
// Returns an error if no match is found or if multiple deployments share the same display name
// (listing the matches so the caller can retry with a deployment ID).
func findDeploymentByDisplayName(deployments []depapi.DeploymentV1Deployment, displayName string) (depapi.DeploymentV1Deployment, error) {
	var matches []depapi.DeploymentV1Deployment
	for _, d := range deployments {
		if d.DisplayName != nil && *d.DisplayName == displayName {
			matches = append(matches, d)
		}
	}
	switch len(matches) {
	case 0:
		return depapi.DeploymentV1Deployment{}, fmt.Errorf("no deployment found with display name %q", displayName)
	case 1:
		return matches[0], nil
	default:
		var sb strings.Builder
		fmt.Fprintf(&sb, "multiple deployments found with display name %q; use a deployment ID instead:\n", displayName)
		for _, m := range matches {
			fmt.Fprintf(&sb, "  display-name: %s  id: %s\n", derefString(m.DisplayName), derefString(m.DeployId))
		}
		return depapi.DeploymentV1Deployment{}, errors.New(strings.TrimRight(sb.String(), "\n"))
	}
}

func runGetDeploymentCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	query := args[0]

	if !isDeploymentID(query) {
		// DisplayName-based lookup: list all deployments and do an exact client-side match.
		resp, err := deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
			&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{},
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving deployments"); err != nil {
			return err
		}
		deployment, err := findDeploymentByDisplayName(resp.JSON200.Deployments, query)
		if err != nil {
			return err
		}
		query = derefString(deployment.DeployId)
	}

	resp, err := deploymentClient.DeploymentV1DeploymentServiceGetDeploymentWithResponse(ctx, projectName, query,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", fmt.Sprintf("error getting deployment %s", query)); !proceed {
		return err
	}
	if err := printDeployments(cmd, writer, &[]depapi.DeploymentV1Deployment{resp.JSON200.Deployment}, nil, nil, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func getValidatedDeploymentFilter(
	ctx context.Context,
	cmd *cobra.Command,
	deploymentClient depapi.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return normalizeFilterWithAPIProbe(raw, "deployments", depapi.DeploymentV1Deployment{}, func(filter string) (bool, error) {
		pageSize := int32(1)
		offset := int32(0)
		resp, err := deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
			&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{
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
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating deployment filter"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func runSetDeploymentCommand(cmd *cobra.Command, args []string) error {
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	deploymentID := args[0]

	if !isDeploymentID(deploymentID) {
		// DisplayName-based lookup: list all deployments and do an exact client-side match.
		resp, err := deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
			&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{},
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving deployments"); err != nil {
			return err
		}
		deployment, err := findDeploymentByDisplayName(resp.JSON200.Deployments, deploymentID)
		if err != nil {
			return err
		}
		deploymentID = derefString(deployment.DeployId)
	}

	overrideValues, err := getOverrideValues(cmd)
	if err != nil {
		return err
	}

	targetClusters, deploymentType, err := getTargetClusters(cmd, true)
	if err != nil {
		return err
	}

	gresp, err := deploymentClient.DeploymentV1DeploymentServiceGetDeploymentWithResponse(ctx, projectName, deploymentID,
		auth.AddAuthHeader)
	if err != nil {
		return err
	}

	if gresp.HTTPResponse.StatusCode != http.StatusOK {
		return checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("error getting for application %s", deploymentID))
	}

	dep := gresp.JSON200.Deployment

	request := depapi.DeploymentV1DeploymentServiceUpdateDeploymentJSONRequestBody{
		DeployId:       &deploymentID,
		Name:           getFlagOrDefault(cmd, "name", dep.Name),
		AppName:        *getFlagOrDefault(cmd, "application-name", &dep.AppName),
		AppVersion:     *getFlagOrDefault(cmd, "application-version", &dep.AppVersion),
		DisplayName:    getFlagOrDefault(cmd, "display-name", dep.DisplayName),
		ProfileName:    getFlagOrDefault(cmd, "profile", dep.ProfileName),
		OverrideValues: dep.OverrideValues,
		TargetClusters: dep.TargetClusters,
	}
	if len(overrideValues) > 0 {
		request.OverrideValues = &overrideValues
	}
	if len(*targetClusters) > 0 {
		request.TargetClusters = targetClusters
		request.DeploymentType = &deploymentType
	}

	resp, err := deploymentClient.DeploymentV1DeploymentServiceUpdateDeploymentWithResponse(cmd.Context(), projectName, deploymentID, request, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error updating deployment %s", deploymentID)); err != nil {
		return err
	}
	fmt.Printf("Deployment '%s' updated successfully\n", deploymentID)
	return nil
}

func runUpgradeDeploymentCommand(cmd *cobra.Command, args []string) error {
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	deploymentID := args[0]
	newPackageVersion, _ := cmd.Flags().GetString("package-version")

	if !isDeploymentID(deploymentID) {
		// DisplayName-based lookup: list all deployments and do an exact client-side match.
		resp, err := deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
			&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{},
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving deployments"); err != nil {
			return err
		}
		deployment, err := findDeploymentByDisplayName(resp.JSON200.Deployments, deploymentID)
		if err != nil {
			return err
		}
		deploymentID = derefString(deployment.DeployId)
	}

	// Get the current deployment to retrieve package name and other details
	gresp, err := deploymentClient.DeploymentV1DeploymentServiceGetDeploymentWithResponse(ctx, projectName, deploymentID,
		auth.AddAuthHeader)
	if err != nil {
		return err
	}

	if gresp.HTTPResponse.StatusCode != http.StatusOK {
		return checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("error getting deployment %s", deploymentID))
	}

	dep := gresp.JSON200.Deployment

	// Build the update request with the new package version
	request := depapi.DeploymentV1DeploymentServiceUpdateDeploymentJSONRequestBody{
		DeployId:       &deploymentID,
		Name:           dep.Name,
		AppName:        dep.AppName,
		AppVersion:     newPackageVersion, // Use the new package version
		DisplayName:    dep.DisplayName,
		ProfileName:    dep.ProfileName,
		OverrideValues: dep.OverrideValues,
		TargetClusters: dep.TargetClusters,
		DeploymentType: dep.DeploymentType,
	}

	resp, err := deploymentClient.DeploymentV1DeploymentServiceUpdateDeploymentWithResponse(cmd.Context(), projectName, deploymentID, request, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error upgrading deployment %s to version %s", deploymentID, newPackageVersion)); err != nil {
		return err
	}
	fmt.Printf("Deployment '%s' upgraded successfully to version '%s'\n", deploymentID, newPackageVersion)
	return nil
}

func runDeleteDeploymentCommand(cmd *cobra.Command, args []string) error {
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	deploymentID := args[0]

	if !isDeploymentID(deploymentID) {
		// DisplayName-based lookup: list all deployments and do an exact client-side match.
		resp, err := deploymentClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
			&depapi.DeploymentV1DeploymentServiceListDeploymentsParams{},
			auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error while retrieving deployments"); err != nil {
			return err
		}
		deployment, err := findDeploymentByDisplayName(resp.JSON200.Deployments, deploymentID)
		if err != nil {
			return err
		}
		deploymentID = derefString(deployment.DeployId)
	}

	resp, err := deploymentClient.DeploymentV1DeploymentServiceDeleteDeploymentWithResponse(ctx, projectName, deploymentID,
		&depapi.DeploymentV1DeploymentServiceDeleteDeploymentParams{}, auth.AddAuthHeader)

	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting deployment %s", deploymentID)); err != nil {
		return err
	}
	fmt.Printf("Deployment '%s' deleted successfully\n", deploymentID)
	return nil
}

// getValidApplicationNames fetches the deployment package and returns the list of valid application names
func getValidApplicationNames(ctx context.Context, catalogClient catapi.ClientWithResponsesInterface, projectName, packageName, packageVersion string) (map[string]bool, error) {
	resp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, packageName, packageVersion, auth.AddAuthHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployment package: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to fetch deployment package %s:%s (status %d)", packageName, packageVersion, resp.StatusCode())
	}

	if resp.JSON200 == nil || len(resp.JSON200.DeploymentPackage.ApplicationReferences) == 0 {
		return nil, fmt.Errorf("deployment package %s:%s has no applications", packageName, packageVersion)
	}

	// Build a map of valid application names
	validNames := make(map[string]bool)
	for _, appRef := range resp.JSON200.DeploymentPackage.ApplicationReferences {
		validNames[appRef.Name] = true
	}

	if len(validNames) == 0 {
		return nil, fmt.Errorf("deployment package %s:%s has no valid application names", packageName, packageVersion)
	}

	return validNames, nil
}

// validateApplicationNames checks that all application names in overrides exist in validNames
func validateApplicationNames(overrides []depapi.DeploymentV1OverrideValues, validNames map[string]bool, flagName string) error {
	var invalidNames []string
	for _, override := range overrides {
		appName := override.AppName
		if !validNames[appName] {
			invalidNames = append(invalidNames, appName)
		}
	}

	if len(invalidNames) > 0 {
		validList := make([]string, 0, len(validNames))
		for name := range validNames {
			validList = append(validList, name)
		}
		sort.Strings(validList)
		return fmt.Errorf("invalid application name(s) in --%s: %v. Valid names: %v",
			flagName, invalidNames, validList)
	}

	return nil
}

// validateTargetClustersApplicationNames checks that all application names in target clusters exist in validNames
func validateTargetClustersApplicationNames(targetClusters *[]depapi.DeploymentV1TargetClusters, validNames map[string]bool) error {
	if targetClusters == nil {
		return nil
	}

	var invalidNames []string
	seenInvalid := make(map[string]bool)

	for _, tc := range *targetClusters {
		if tc.AppName != nil {
			appName := *tc.AppName
			if !validNames[appName] && !seenInvalid[appName] {
				invalidNames = append(invalidNames, appName)
				seenInvalid[appName] = true
			}
		}
	}

	if len(invalidNames) > 0 {
		validList := make([]string, 0, len(validNames))
		for name := range validNames {
			validList = append(validList, name)
		}
		sort.Strings(validList)
		return fmt.Errorf("invalid application name(s) in --application-cluster-id or --application-label: %v. Valid names: %v",
			invalidNames, validList)
	}

	return nil
}
