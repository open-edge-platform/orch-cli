// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"

	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_DEPLOYMENT_PROFILE_FORMAT         = "table{{.Name}}\t{{.DisplayName}}\t{{.Description}}\t{{len .ApplicationProfiles}}"
	DEFAULT_DEPLOYMENT_PROFILE_INSPECT_FORMAT = `Name: {{.Name}}
Display Name: {{str .DisplayName}}
Description: {{str .Description}}
Profiles:{{range $app, $profile := .ApplicationProfiles}}
  {{$app}}:{{$profile}}{{- end}}
Create Time: {{.CreateTime}}
Update Time: {{.UpdateTime}}
`
	DEPLOYMENT_PROFILE_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_DEPLOYMENT_PROFILE_OUTPUT_TEMPLATE"
)

func getCreateDeploymentProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package-profile <deployment-package-name> <version> <name> [flags]",
		Aliases: deploymentProfileAliases,
		Short:   "Create a deployment package profile",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli create deployment-package-profile my-deployment-package 1.0.0 my-profile --display-name 'My Profile' --description 'This is my profile' --project my-project",
		RunE:    runCreateDeploymentProfileCommand,
	}
	addEntityFlags(cmd, "deployment-package-profile")
	cmd.Flags().StringToString("application-profile", map[string]string{}, "application name to application profile bindings")
	return cmd
}

func getListDeploymentProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package-profiles <deployment-package-name> <version> [flags]",
		Aliases: deploymentProfileAliases,
		Short:   "List all deployment package profiles",
		Args:    cobra.ExactArgs(2),
		Example: "orch-cli list deployment-package-profiles my-deployment-package 1.0.0 --project my-project",
		RunE:    runListDeploymentProfilesCommand,
	}
	// annotations removed: dynamic header-derived hints will be used instead
	addStandardListOutputFlags(cmd)
	return cmd
}

func getGetDeploymentProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package-profile <deployment-package-name> <version> <name> [flags]",
		Aliases: deploymentProfileAliases,
		Short:   "Get a deployment profile",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli get deployment-package-profile my-deployment-package 1.0.0 my-profile --project my-project",
		RunE:    runGetDeploymentProfileCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getSetDeploymentProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package-profile <deployment-package-name> <version> <name> [flags]",
		Aliases: deploymentProfileAliases,
		Short:   "Update a deployment package profile",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli set deployment-package-profile my-deployment-package 1.0.0 my-profile --display-name 'My Updated Profile' --description 'This is my updated profile' --application-profile app1=profile1,app2=profile2 --project my-project",
		RunE:    runSetDeploymentProfileCommand,
	}
	addEntityFlags(cmd, "deployment-package-profile")
	cmd.Flags().StringToString("application-profile", map[string]string{}, "application name to application profile bindings")
	return cmd
}

func getDeleteDeploymentProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment-package-profile <deployment-package-name> <version> <name> [flags]",
		Aliases: deploymentProfileAliases,
		Short:   "Delete an application profile",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli delete deployment-package-profile my-deployment-package 1.0.0 my-profile --project my-project",
		RunE:    runDeleteDeploymentProfileCommand,
	}
	return cmd
}

func getDeploymentProfileOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_DEPLOYMENT_PROFILE_INSPECT_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_DEPLOYMENT_PROFILE_FORMAT, DEPLOYMENT_PROFILE_OUTPUT_TEMPLATE_ENVVAR)
}

func printDeploymentProfiles(cmd *cobra.Command, writer io.Writer, profileList *[]catapi.CatalogV3DeploymentProfile, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getDeploymentProfileOutputFormat(cmd, verbose)
	if err != nil {
		return err
	}

	filterSpec := ""
	if outputType == "table" && outputFilter != nil && *outputFilter != "" {
		filterSpec = *outputFilter
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      *profileList,
	}

	GenerateOutput(writer, &result)
	return nil
}

func runCreateDeploymentProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]
	profileName := args[2]

	applicationProfiles, err := cmd.Flags().GetStringToString("application-profile")
	if err != nil {
		return fmt.Errorf("error getting application profiles from flags: %w", err)
	}

	gresp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
		return err
	}

	pkg := gresp.JSON200.DeploymentPackage
	profile := catapi.CatalogV3DeploymentProfile{
		Name:                profileName,
		DisplayName:         &displayName,
		Description:         &description,
		ApplicationProfiles: applicationProfiles,
	}
	profiles := *pkg.Profiles

	// Check if a profile with this name already exists
	for _, existingProfile := range profiles {
		if existingProfile.Name == profileName {
			return fmt.Errorf("deployment profile %s already exists for deployment package %s:%s", profileName, name, version)
		}
	}

	// Insert the new profile, keeping the implicit default profile if it exists
	profiles = append(profiles, profile)

	// Set default profile name only if there isn't one already or if it's the implicit default
	if pkg.DefaultProfileName == nil || *pkg.DefaultProfileName == "" || *pkg.DefaultProfileName == "deployment-profile-1" || *pkg.DefaultProfileName == "default-profile" {
		pkg.DefaultProfileName = &profileName
	}

	pkg.Profiles = &profiles

	resp, err := updateDeploymentPackage(ctx, projectName, catalogClient, pkg)
	if err != nil {
		return err
	}

	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating deployment profile %s", profileName)); err != nil {
		return err
	}
	fmt.Printf("Deployment profile '%s' created successfully for deployment package '%s:%s'\n", profileName, name, version)
	return nil
}

func runListDeploymentProfilesCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]

	resp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		fmt.Sprintf("error listing deployment profiles for deployment package %s:%s", name, version)); !proceed {
		return err
	}
	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printDeploymentProfiles(cmd, writer, resp.JSON200.DeploymentPackage.Profiles, &outputFilter, verbose); err != nil {
		return err
	}
	return writer.Flush()
}

func runGetDeploymentProfileCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]
	profileName := args[2]

	resp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true, "",
		fmt.Sprintf("error getting deployment profile %s for deployment package %s:%s", profileName, name, version)); !proceed {
		return err
	}

	pkg := resp.JSON200.DeploymentPackage
	for _, profile := range *pkg.Profiles {
		if profile.Name == profileName {
			if err := printDeploymentProfiles(cmd, writer, &[]catapi.CatalogV3DeploymentProfile{profile}, nil, verbose); err != nil {
				return err
			}
			return writer.Flush()
		}
	}
	return errors.NewNotFound("deployment profile %s for deployment package %s:%s not found", profileName, name, version)
}

func runSetDeploymentProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]
	profileName := args[2]

	gresp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
		return err
	}

	pkg := gresp.JSON200.DeploymentPackage
	profiles := *pkg.Profiles

	// Find our deployment profile
	ok := false
	for i, profile := range profiles {
		if profile.Name == profileName {
			profile.DisplayName = getFlagOrDefault(cmd, "display-name", profile.DisplayName)
			profile.Description = getFlagOrDefault(cmd, "description", profile.Description)
			newApplicationProfiles, _ := cmd.Flags().GetStringToString("application-profile")
			if len(newApplicationProfiles) > 0 {
				profile.ApplicationProfiles = newApplicationProfiles
			}
			profiles[i] = profile
			ok = true
			break
		}
	}
	if !ok {
		return errors.NewNotFound("deployment profile %s not found for deployment package %s:%s", profileName, name, version)
	}

	pkg.Profiles = &profiles

	resp, err := updateDeploymentPackage(ctx, projectName, catalogClient, pkg)
	if err != nil {
		return err
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while updating deployment profile %s for deployment package %s:%s",
		profileName, name, version)); err != nil {
		return err
	}
	fmt.Printf("Deployment profile '%s' updated successfully for deployment package '%s:%s'\n", profileName, name, version)
	return nil
}

func runDeleteDeploymentProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]
	profileName := args[2]

	gresp, err := catalogClient.CatalogServiceGetDeploymentPackageWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
		return err
	}

	pkg := gresp.JSON200.DeploymentPackage
	profiles := *pkg.Profiles

	// Remove the profile from the list
	for i, profile := range profiles {
		if profile.Name == profileName {
			profiles = append(profiles[:i], profiles[i+1:]...)
			break
		}
	}

	// Adjust default profile name as needed
	if pkg.DefaultProfileName != nil && *pkg.DefaultProfileName == profileName {
		if len(profiles) > 0 {
			pkg.DefaultProfileName = &profiles[0].Name
		} else {
			pkg.DefaultProfileName = nil
		}
	}

	pkg.Profiles = &profiles

	resp, err := updateDeploymentPackage(ctx, projectName, catalogClient, pkg)
	if err != nil {
		return err
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting deployment profile %s for deployment package %s:%s",
		profileName, name, version)); err != nil {
		return err
	}
	fmt.Printf("Deployment profile '%s' deleted successfully from deployment package '%s:%s'\n", profileName, name, version)
	return nil
}
