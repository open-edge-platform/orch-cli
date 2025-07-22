// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	b64 "encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/spf13/cobra"
)

func getCreateProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile <application-name> <version> <name> [flags]",
		Short:   "Create an application profile",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli create profile my-app 1.0.0 my-profile --display-name 'My Profile' --description 'This is my profile' --chart-values values.yaml --project my-project",
		RunE:    runCreateProfileCommand,
	}
	addEntityFlags(cmd, "profile")
	cmd.Flags().String("chart-values", "-", "path to the values.yaml file; - for stdin")
	return cmd
}

func getListProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profiles <application-name> <version> [flags]",
		Short:   "List all application profiles",
		Example: "orch-cli list profiles my-app 1.0.0 --project my-project",
		Args:    cobra.ExactArgs(2),
		RunE:    runListProfilesCommand,
	}
	return cmd
}

func getGetProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile <application-name> <version> <name> [flags]",
		Short:   "Get an application profile",
		Example: "orch-cli get profile my-app 1.0.0 my-profile --project my-project",
		Args:    cobra.ExactArgs(3),
		RunE:    runGetProfileCommand,
	}
	return cmd
}

func getSetProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile <application-name> <version> <name> [flags]",
		Short:   "Update an application profile",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli set profile my-app 1.0.0 my-profile --display-name 'Updated Profile' --description 'Updated description' --chart-values new-values.yaml --project my-project",
		RunE:    runSetProfileCommand,
	}
	addEntityFlags(cmd, "profile")
	cmd.Flags().String("chart-values", "", "path to the values.yaml file; - for stdin")
	return cmd
}

func getDeleteProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile <application-name> <version> <name> [flags]",
		Short:   "Delete an application profile",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli delete profile my-app 1.0.0 my-profile --project my-project",
		RunE:    runDeleteProfileCommand,
	}
	return cmd
}

var profileHeader = fmt.Sprintf("%s\t%s\t%s", "Name", "Display Name", "Description")

func printProfiles(writer io.Writer, profileList *[]catapi.Profile, verbose bool) {
	for _, p := range *profileList {
		if !verbose {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", p.Name, valueOrNone(p.DisplayName), valueOrNone(p.Description))
		} else {
			_, _ = fmt.Fprintf(writer, "Name: %s\n", p.Name)
			_, _ = fmt.Fprintf(writer, "Display Name: %s\n", valueOrNone(p.DisplayName))
			_, _ = fmt.Fprintf(writer, "Description: %s\n", valueOrNone(p.Description))

			if len(*p.DeploymentRequirement) != 0 {
				requirements := make([]string, 0, len(*p.DeploymentRequirement))
				for _, dr := range *p.DeploymentRequirement {
					requirements = append(requirements, fmt.Sprintf("%s:%s", dr.Name, dr.Version))
				}
				_, _ = fmt.Fprintf(writer, "Deployment Requirements: %s\n", requirements)
			}

			_, _ = fmt.Fprintf(writer, "Create Time: %s\n", p.CreateTime.Format(timeLayout))
			_, _ = fmt.Fprintf(writer, "Update Time: %s\n\n", p.UpdateTime.Format(timeLayout))
			if len(*p.ParameterTemplates) != 0 {
				_, _ = fmt.Fprintf(writer, "Parameter templates:\n")
				for _, pt := range *p.ParameterTemplates {
					_, _ = fmt.Fprintf(writer, "   Name: %s\n   Type: %s\n   Display Name: %s\n   Default: %s\n   Suggested values: %s\n\n",
						pt.Name, pt.Type, valueOrNone(pt.DisplayName), *pt.Default, strings.Join((*pt.SuggestedValues)[:], ","))
				}
			}

			if p.ChartValues != nil && *p.ChartValues != "" {
				_, _ = fmt.Fprintf(writer, "Chart Values:\n")
				decodedValues, err := b64.StdEncoding.DecodeString(*p.ChartValues)

				if err == nil {
					lines := strings.Split(string(decodedValues), "\n")
					for _, line := range lines {
						_, _ = fmt.Fprintf(writer, "  %s\n", line)
					}
				} else {
					_, _ = fmt.Fprintf(writer, "  [Error decoding chart values: %v]\n", err)
					// If decoding fails, show the raw encoded data
					_, _ = fmt.Fprintf(writer, "  Raw encoded data: %s\n", *p.ChartValues)
				}
				_, _ = fmt.Fprintf(writer, "\n")
			}
		}
	}
}

func runCreateProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
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

	chartBytes, err := readInput(*getFlag(cmd, "chart-values"))
	if err != nil {
		return fmt.Errorf("error reading values.yaml content: %w", err)
	}

	chartValues := b64.StdEncoding.EncodeToString(chartBytes)

	gresp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
		return err
	}

	application := gresp.JSON200.Application
	profiles := append(*application.Profiles, catapi.Profile{
		Name:        profileName,
		DisplayName: &displayName,
		Description: &description,
		ChartValues: &chartValues,
	})

	if application.DefaultProfileName == nil || *application.DefaultProfileName == "" {
		application.DefaultProfileName = &profileName
	}

	resp, err := catalogClient.CatalogServiceUpdateApplicationWithResponse(ctx, projectName, name, version,
		catapi.CatalogServiceUpdateApplicationJSONRequestBody{
			Name:               name,
			Version:            version,
			DisplayName:        application.DisplayName,
			Description:        application.Description,
			ChartName:          application.ChartName,
			ChartVersion:       application.ChartVersion,
			HelmRegistryName:   application.HelmRegistryName,
			ImageRegistryName:  application.ImageRegistryName,
			Profiles:           &profiles,
			DefaultProfileName: application.DefaultProfileName,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error creating profile %s of application %s:%s",
		profileName, name, version))
}

func runListProfilesCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]

	resp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, profileHeader,
		fmt.Sprintf("error listing profiles for application %s:%s", name, version)); !proceed {
		return err
	}
	printProfiles(writer, resp.JSON200.Application.Profiles, verbose)
	return writer.Flush()
}

func runGetProfileCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]
	profileName := args[2]

	resp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, profileHeader,
		fmt.Sprintf("error listing profiles for application %s:%s", name, version)); !proceed {
		return err
	}

	for _, profile := range *resp.JSON200.Application.Profiles {
		if profile.Name == profileName {
			printProfiles(writer, &[]catapi.Profile{profile}, verbose)
			return writer.Flush()
		}
	}
	return errors.NewNotFound("profile %s for application %s:%s not found", profileName, name, version)
}

func runSetProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]
	profileName := args[2]

	gresp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
		return err
	}

	// Scan through the profiles and update the named profile
	application := gresp.JSON200.Application
	profiles := *application.Profiles

	var profile *catapi.Profile
	for i, p := range profiles {
		if p.Name == profileName {
			profile = &profiles[i]
			break
		}
	}

	// If this is the first profile being added, let's also set it as the default profile name
	if len(profiles) == 1 {
		application.DefaultProfileName = &profileName
	}

	profile.DisplayName = getFlagOrDefault(cmd, "display-name", profile.DisplayName)
	profile.Description = getFlagOrDefault(cmd, "description", profile.Description)

	// If the chart-values flag was given, fetch the new content to replace the existing one
	newChartValuesPath := *getFlag(cmd, "chart-values")
	if len(newChartValuesPath) > 0 {
		chartValueBytes, err := readInput(*getFlag(cmd, "chart-values"))
		if err != nil {
			return fmt.Errorf("error reading chart-values content: %w", err)
		}
		newChartValues := b64.StdEncoding.EncodeToString(chartValueBytes)
		profile.ChartValues = &newChartValues
	}

	resp, err := catalogClient.CatalogServiceUpdateApplicationWithResponse(ctx, projectName, name, version,
		catapi.CatalogServiceUpdateApplicationJSONRequestBody{
			Name:               name,
			Version:            version,
			DisplayName:        application.DisplayName,
			Description:        application.Description,
			ChartName:          application.ChartName,
			ChartVersion:       application.ChartVersion,
			HelmRegistryName:   application.HelmRegistryName,
			ImageRegistryName:  application.ImageRegistryName,
			Profiles:           &profiles,
			DefaultProfileName: application.DefaultProfileName,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error updating profile %s of application %s:%s",
		profileName, name, version))
}

func runDeleteProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := getCatalogServiceContext(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	version := args[1]
	profileName := args[2]

	gresp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
		return err
	}

	// Scan through the profiles and eliminate the named profile
	application := gresp.JSON200.Application
	profiles := *application.Profiles
	for i, profile := range profiles {
		if profile.Name == profileName {
			profiles = append(profiles[:i], profiles[i+1:]...)
			break
		}
	}

	if application.DefaultProfileName != nil && *application.DefaultProfileName == profileName {
		if len(profiles) == 0 {
			application.DefaultProfileName = nil
		} else {
			application.DefaultProfileName = profiles[0].DisplayName
		}
	}

	resp, err := catalogClient.CatalogServiceUpdateApplicationWithResponse(ctx, projectName, name, version,
		catapi.CatalogServiceUpdateApplicationJSONRequestBody{
			Name:               name,
			Version:            version,
			DisplayName:        application.DisplayName,
			Description:        application.Description,
			ChartName:          application.ChartName,
			ChartVersion:       application.ChartVersion,
			HelmRegistryName:   application.HelmRegistryName,
			ImageRegistryName:  application.ImageRegistryName,
			Profiles:           &profiles,
			DefaultProfileName: application.DefaultProfileName,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting profile %s of application %s:%s",
		profileName, name, version))
}
