// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"

	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
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
		Aliases: []string{"deployment-profiles", "package-profiles", "bundle-profiles"},
		Short:   "List all deployment package profiles",
		Args:    cobra.ExactArgs(2),
		Example: "orch-cli list deployment-package-profiles my-deployment-package 1.0.0 --project my-project",
		RunE:    runListDeploymentProfilesCommand,
	}
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

var deploymentProfileHeader = fmt.Sprintf("%s\t%s\t%s\t%s", "Name", "Display Name", "Description", "Profile Count")

func printDeploymentProfiles(writer io.Writer, profileList *[]catapi.DeploymentProfile, verbose bool) {
	for _, p := range *profileList {
		if !verbose {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%d\n", p.Name,
				valueOrNone(p.DisplayName), valueOrNone(p.Description), len(p.ApplicationProfiles))
		} else {
			_, _ = fmt.Fprintf(writer, "Name: %s\n", p.Name)
			_, _ = fmt.Fprintf(writer, "Display Name: %s\n", valueOrNone(p.DisplayName))
			_, _ = fmt.Fprintf(writer, "Description: %s\n", valueOrNone(p.Description))
			_, _ = fmt.Fprintf(writer, "Profiles: %s\n", p.ApplicationProfiles)
			_, _ = fmt.Fprintf(writer, "Create Time: %s\n", p.CreateTime.Format(timeLayout))
			_, _ = fmt.Fprintf(writer, "Update Time: %s\n\n", p.UpdateTime.Format(timeLayout))
		}
	}
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
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
		return err
	}

	pkg := gresp.JSON200.DeploymentPackage
	profile := catapi.DeploymentProfile{
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
	if pkg.DefaultProfileName == nil || *pkg.DefaultProfileName == "" || *pkg.DefaultProfileName == "implicit-default" {
		pkg.DefaultProfileName = &profileName
	}

	pkg.Profiles = &profiles

	resp, err := updateDeploymentPackage(ctx, projectName, catalogClient, pkg)
	if err != nil {
		return err
	}
	
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating deployment profile %s", profileName))
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
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, deploymentProfileHeader,
		fmt.Sprintf("error listing deployment profiles for deployment package %s:%s", name, version)); !proceed {
		return err
	}

	printDeploymentProfiles(writer, resp.JSON200.DeploymentPackage.Profiles, verbose)
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
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, deploymentProfileHeader,
		fmt.Sprintf("error getting deployment profile %s for deployment package %s:%s", profileName, name, version)); !proceed {
		return err
	}

	pkg := resp.JSON200.DeploymentPackage
	for _, profile := range *pkg.Profiles {
		if profile.Name == profileName {
			printDeploymentProfiles(writer, &[]catapi.DeploymentProfile{profile}, verbose)
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
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
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
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while updating deployment profile %s for deployment package %s:%s",
		profileName, name, version))
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
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("deployment package %s:%s not found", name, version)); err != nil {
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
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting deployment profile %s for deployment package %s:%s",
		profileName, name, version))
}
