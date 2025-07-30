// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/open-edge-platform/cli/pkg/auth"

	depapi "github.com/open-edge-platform/cli/pkg/rest/deployment"
	"github.com/spf13/cobra"
)

func getCreateDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <application-name> <version> [flags]",
		Short:   "Create a deployment",
		Example: "orch-cli create deployment my-package 1.0.0 --project sample-project --display-name my-deployment --profile sample-profile --application-label <app>.<label>=<label-value>",
		Args:    cobra.ExactArgs(2),

		RunE: runCreateDeploymentCommand,
	}
	cmd.Flags().String("display-name", "", "deployment display name")
	cmd.Flags().String("profile", "", "deployment profile to use")
	cmd.Flags().StringToString("application-namespace", map[string]string{}, "application target namespaces in format '<app>=<namespace>'")
	cmd.Flags().StringToString("application-set", map[string]string{}, "application set value overrides in form of '<app>.<prop>=<prop-value>'")
	cmd.Flags().StringToString("application-label", map[string]string{}, "automatic deployment of application to clusters in the form of '<app>.<label>=<label-value>'")
	cmd.Flags().StringToString("application-cluster-id", map[string]string{}, "manunal deployment of application to clusters in the form of '<app>=<cluster-id>'")
	return cmd
}

func getListDeploymentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployments [flags]",
		Aliases: []string{"deployment"},
		Short:   "List all deployments",
		Example: "orch-cli list deployments --project some-project",
		RunE:    runListDeploymentsCommand,
	}
	return cmd
}

func getGetDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <deployment-id> [flags]",
		Short:   "Get a deployment",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli get deployment 12345 --project some-project",
		RunE:    runGetDeploymentCommand,
	}
	return cmd
}

func getSetDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <deployment-id> [flags]",
		Short:   "Update a deployment",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli set deployment 12345 --project some-project --name my-deployment --package-name my-package --package-version 1.0.0 --profile sample-profile --application-namespace <app>=<namespace> --application-set <app>.<prop>=<prop-value> --application-label <app>.<label>=<label-value>",
		RunE:    runSetDeploymentCommand,
	}
	cmd.Flags().String("name", "", "deployment name")
	cmd.Flags().String("package-name", "", "deployment package name")
	cmd.Flags().String("package-version", "", "deployment package version")
	cmd.Flags().String("profile", "", "deployment profileto use")
	cmd.Flags().StringToString("application-namespace", map[string]string{}, "application target namespaces in format '<app>=<namespace>'")
	cmd.Flags().StringToString("application-set", map[string]string{}, "application set value overrides in form of '<app>.<prop>=<prop-value>'")
	cmd.Flags().StringToString("application-label", map[string]string{}, "automatic deployment of application to clusters in the form of '<app>.<label>=<label-value>'")
	cmd.Flags().StringToString("application-cluster-id", map[string]string{}, "manual deployment of application to clusters in the form of '<app>=<cluster-id>'")
	return cmd
}

func getDeleteDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <deployment-id> [flags]",
		Short:   "Delete a deployment",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli delete deployment 12345 --project some-project",
		RunE:    runDeleteDeploymentCommand,
	}
	return cmd
}

var deploymentHeader = fmt.Sprintf("%s\t%s\t%s\t%s\t%s",
	"Deployment ID", "Name", "Display Name", "Profile", "State")

func printDeployments(writer *tabwriter.Writer, deployments *[]depapi.Deployment, verbose bool) {
	for _, d := range *deployments {
		if !verbose {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", *d.DeployId, *d.Name,
				*d.DisplayName, *d.ProfileName, *d.Status.State)
		} else {
			_, _ = fmt.Fprintf(writer, "Deployment ID: %s\n", *d.DeployId)
			_, _ = fmt.Fprintf(writer, "Name: %s\n", *d.Name)
			_, _ = fmt.Fprintf(writer, "Display Name: %s\n", *d.DisplayName)
			_, _ = fmt.Fprintf(writer, "Profile: %s\n", *d.ProfileName)
			_, _ = fmt.Fprintf(writer, "State: %s\n", *d.Status.State)
			// FIXME: add the rest
			_, _ = fmt.Fprintf(writer, "Create Time: %s\n", d.CreateTime.Format(timeLayout))
		}
	}
}

func runCreateDeploymentCommand(cmd *cobra.Command, args []string) error {
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	appName := args[0]
	appVersion := args[1]

	overrideValues, err := getOverrideValues(cmd)
	if err != nil {
		return err
	}

	targetClusters, deploymentType, err := getTargetClusters(cmd, false) // do not allow empty target clusters
	if err != nil {
		return err
	}

	resp, err := deploymentClient.DeploymentServiceCreateDeploymentWithResponse(ctx, projectName,
		depapi.DeploymentServiceCreateDeploymentJSONRequestBody{
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
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error creating deployment for application %s:%s", appName, appVersion))
}

func getOverrideValues(cmd *cobra.Command) ([]depapi.OverrideValues, error) {
	namespaces, _ := cmd.Flags().GetStringToString("application-namespace")
	sets, _ := cmd.Flags().GetStringToString("application-set")
	return getOverrideValuesRaw(namespaces, sets)
}

func getOverrideValuesRaw(namespaces, sets map[string]string) ([]depapi.OverrideValues, error) {
	overrides := make(map[string]depapi.OverrideValues, 0)

	// First, accumulate any app target namespace settings in the format "<app-name>=<target-namespace>"
	for app, namespace := range namespaces {
		ns := namespace
		values := make(map[string]interface{}, 0)
		overrides[app] = depapi.OverrideValues{AppName: app, TargetNamespace: &ns, Values: &values}
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
			values := make(map[string]interface{}, 0)
			override = depapi.OverrideValues{AppName: app, Values: &values}
		}
		override.Values = addProperty(override.Values, app, prop, value)
		overrides[app] = override
	}

	// Transform overrides map into array
	overrideValues := make([]depapi.OverrideValues, 0)
	for _, override := range overrides {
		overrideValues = append(overrideValues, override)
	}
	return overrideValues, nil
}

func addProperty(values *map[string]interface{}, app string, prop string, value string) *map[string]interface{} {
	ovs := *values
	segs := strings.SplitN(prop, ".", 2)
	if len(segs) == 1 {
		ovs[segs[0]] = parseValue(value)
	} else if len(segs) > 1 {
		gprops, ok := ovs[segs[0]]
		if !ok {
			newProps := make(map[string]interface{}, 0)
			gprops = &newProps
		}
		props := gprops.(*map[string]interface{})
		ovs[segs[0]] = addProperty(props, app, segs[1], value)
	}
	return values
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

func getTargetClusters(cmd *cobra.Command, allowEmpty bool) (*[]depapi.TargetClusters, string, error) {
	targetClustersByLabel, err := getTargetClustersByLabel(cmd)
	if err != nil {
		return nil, "", err
	}

	targetClustersByID, err := getTargetClustersByID(cmd)
	if err != nil {
		return nil, "", err
	}

	if targetClustersByLabel != nil && len(*targetClustersByLabel) > 0 {
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
	return &[]depapi.TargetClusters{}, "", nil
}

func getTargetClustersByLabel(cmd *cobra.Command) (*[]depapi.TargetClusters, error) {
	// Accumulate any app target cluster labels in format "<app-name>.<label-name>=<label-value>"
	targets := make(map[string]depapi.TargetClusters, 0)
	labels, _ := cmd.Flags().GetStringToString("application-label")
	for appLabel, value := range labels {
		fields := strings.SplitN(appLabel, ".", 2)
		if len(fields) < 2 {
			return nil, fmt.Errorf("label %s not in format <app-name>.<label-name>", appLabel)
		}
		app := fields[0]
		label := fields[1]

		target, ok := targets[app]
		lbls := make(map[string]string, 1)
		lbls[label] = value

		if ok {
			// NOTE: Expect ok can never be true because GetStringToString overwrites existing map keys
			// TODO: Support multiple labels for the same app
			return nil, fmt.Errorf("application %s already has a target cluster: %+v", app, target)
		}

		target = depapi.TargetClusters{AppName: &app, Labels: &lbls}
		targets[app] = target
	}

	// Transform targets map into array
	targetClusters := make([]depapi.TargetClusters, 0, len(targets))
	for _, target := range targets {
		targetClusters = append(targetClusters, target)
	}
	return &targetClusters, nil
}

func getTargetClustersByID(cmd *cobra.Command) (*[]depapi.TargetClusters, error) {
	// Accumulate any app target cluster IDs in format "<app-name>=<cluster-id>""
	targets := make(map[string]depapi.TargetClusters, 0)
	clusterIDs, _ := cmd.Flags().GetStringToString("application-cluster-id")
	for app, clusterID := range clusterIDs {
		target, ok := targets[app]

		if ok {
			// NOTE: Expect ok can never be true because GetStringToString overwrites existing map keys
			// TODO: Support multiple clusters for the same app
			return nil, fmt.Errorf("application %s already has a target cluster: %+v", app, target)
		}

		target = depapi.TargetClusters{AppName: &app, ClusterId: &clusterID}
		targets[app] = target
	}

	// Transform targets map into array
	targetClusters := make([]depapi.TargetClusters, 0, len(targets))
	for _, target := range targets {
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

	resp, err := deploymentClient.DeploymentServiceListDeploymentsWithResponse(ctx, projectName,
		&depapi.DeploymentServiceListDeploymentsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		deploymentHeader, "error getting deployments"); !proceed {
		return err
	}
	printDeployments(writer, &resp.JSON200.Deployments, verbose)
	return writer.Flush()
}

func runGetDeploymentCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	deploymentID := args[0]

	resp, err := deploymentClient.DeploymentServiceGetDeploymentWithResponse(ctx, projectName, deploymentID,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		deploymentHeader, fmt.Sprintf("error getting deployment %s", deploymentID)); !proceed {
		return err
	}
	printDeployments(writer, &[]depapi.Deployment{resp.JSON200.Deployment}, verbose)
	return writer.Flush()
}

func runSetDeploymentCommand(cmd *cobra.Command, args []string) error {
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	deploymentID := args[0]

	overrideValues, err := getOverrideValues(cmd)
	if err != nil {
		return err
	}

	targetClusters, deploymentType, err := getTargetClusters(cmd, true)
	if err != nil {
		return err
	}

	gresp, err := deploymentClient.DeploymentServiceGetDeploymentWithResponse(ctx, projectName, deploymentID,
		auth.AddAuthHeader)
	if err != nil {
		return err
	}

	if gresp.HTTPResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("error getting deployment %s: %s", deploymentID, gresp.HTTPResponse.Status)
	}

	dep := gresp.JSON200.Deployment

	request := depapi.DeploymentServiceUpdateDeploymentJSONRequestBody{
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

	resp, err := deploymentClient.DeploymentServiceUpdateDeploymentWithResponse(cmd.Context(), projectName, deploymentID, request, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error updating deployment %s", deploymentID))
}

func runDeleteDeploymentCommand(cmd *cobra.Command, args []string) error {
	ctx, deploymentClient, projectName, err := DeploymentFactory(cmd)
	if err != nil {
		return err
	}

	deploymentID := args[0]

	resp, err := deploymentClient.DeploymentServiceDeleteDeploymentWithResponse(ctx, projectName, deploymentID,
		&depapi.DeploymentServiceDeleteDeploymentParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting deployment %s", deploymentID))
}
