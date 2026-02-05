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
	"gopkg.in/yaml.v2"
)

func getCreateProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile <application-name> <version> <name> [flags]",
		Short:   "Create an application profile",
		Args:    cobra.ExactArgs(3),
		Aliases: profileAliases,
		Example: "orch-cli create profile my-app 1.0.0 my-profile --display-name 'My Profile' --description 'This is my profile' --chart-values values.yaml --parameter-template env.HOST_IP=string:\"IP address of the target Edge Node\":\"\" --parameter-template env.MINIO_ACCESS_KEY=password:\"Minio access key\":\"\" --project my-project",
		RunE:    runCreateProfileCommand,
	}
	addEntityFlags(cmd, "profile")
	cmd.Flags().String("chart-values", "", "path to the values.yaml file; - for stdin (optional)")
	cmd.Flags().StringSlice("parameter-template", []string{}, "parameter templates in format '<name>=<type>:<display-name>:<default-value>' (types: string, integer)")
	return cmd
}

func getListProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profiles <application-name> <version> [flags]",
		Short:   "List all application profiles",
		Example: "orch-cli list profiles my-app 1.0.0 --project my-project",
		Args:    cobra.ExactArgs(2),
		Aliases: profileAliases,
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
		Aliases: profileAliases,
		RunE:    runGetProfileCommand,
	}
	return cmd
}

func getSetProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile <application-name> <version> <name> [flags]",
		Short:   "Update an application profile",
		Args:    cobra.ExactArgs(3),
		Aliases: profileAliases,
		Example: "orch-cli set profile my-app 1.0.0 my-profile --display-name 'Updated Profile' --description 'Updated description' --chart-values new-values.yaml --parameter-template env.HOST_IP=string:\"IP address\":\"127.0.0.1\" --project my-project",
		RunE:    runSetProfileCommand,
	}
	addEntityFlags(cmd, "profile")
	cmd.Flags().String("chart-values", "", "path to the values.yaml file; - for stdin")
	cmd.Flags().StringSlice("parameter-template", []string{}, "parameter templates in format '<name>=<type>:<display-name>:<default-value>' (types: string, integer)")
	return cmd
}

func getDeleteProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile <application-name> <version> <name> [flags]",
		Short:   "Delete an application profile",
		Args:    cobra.ExactArgs(3),
		Example: "orch-cli delete profile my-app 1.0.0 my-profile --project my-project",
		Aliases: profileAliases,
		RunE:    runDeleteProfileCommand,
	}
	return cmd
}

var profileHeader = fmt.Sprintf("%s\t%s\t%s", "Name", "Display Name", "Description")

// parseParameterTemplates parses parameter template flags from CLI
func parseParameterTemplates(cmd *cobra.Command) (*[]catapi.CatalogV3ParameterTemplate, error) {
	templateSpecs, _ := cmd.Flags().GetStringSlice("parameter-template")

	if len(templateSpecs) == 0 {
		return &[]catapi.CatalogV3ParameterTemplate{}, nil
	}

	var templates []catapi.CatalogV3ParameterTemplate

	for _, spec := range templateSpecs {
		template, err := parseParameterTemplate(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid parameter template '%s': %w", spec, err)
		}
		templates = append(templates, *template)
	}

	return &templates, nil
}

// parseParameterTemplate parses a single parameter template spec: "name=type:display:default"
func parseParameterTemplate(spec string) (*catapi.CatalogV3ParameterTemplate, error) {
	// Split on = to get name and rest
	parts := strings.SplitN(spec, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("format should be 'name=type:display:default'")
	}

	name := strings.TrimSpace(parts[0])
	if name == "" {
		return nil, fmt.Errorf("parameter name cannot be empty")
	}

	// Split the rest on : to get type, display, default
	valueParts := strings.SplitN(parts[1], ":", 3)
	if len(valueParts) != 3 {
		return nil, fmt.Errorf("value format should be 'type:display:default'")
	}

	paramType := strings.TrimSpace(valueParts[0])
	displayName := strings.TrimSpace(valueParts[1])
	defaultValue := strings.TrimSpace(valueParts[2])

	// Validate parameter type
	validTypes := map[string]bool{
		"string":  true,
		"integer": true,
	}

	if !validTypes[paramType] {
		return nil, fmt.Errorf("invalid parameter type '%s', must be one of: string, integer", paramType)
	}

	// Remove quotes from display name and default if present
	displayName = strings.Trim(displayName, "\"'")
	defaultValue = strings.Trim(defaultValue, "\"'")

	template := &catapi.CatalogV3ParameterTemplate{
		Name:            name,
		Type:            paramType,
		DisplayName:     &displayName,
		Default:         &defaultValue,
		SuggestedValues: &[]string{},
	}

	return template, nil
}

func printProfiles(writer io.Writer, profileList *[]catapi.CatalogV3Profile, verbose bool) {
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
				var chartContent string

				if err == nil {
					// Successfully decoded, use decoded content
					chartContent = string(decodedValues)
				} else {
					// Not base64 encoded, use as-is
					chartContent = *p.ChartValues
				}

				lines := strings.Split(chartContent, "\n")
				for _, line := range lines {
					_, _ = fmt.Fprintf(writer, "  %s\n", line)
				}
				_, _ = fmt.Fprintf(writer, "\n")
			}
		}
	}
}

func runCreateProfileCommand(cmd *cobra.Command, args []string) error {
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

	// Read chart values only if provided
	var chartValues string
	chartValuesPath, _ := cmd.Flags().GetString("chart-values")
	if chartValuesPath != "" {
		chartBytes, err := readInputWithLimit(chartValuesPath)
		if err != nil {
			return fmt.Errorf("error reading values.yaml content: %w", err)
		}
		if err := validateValuesYAML(chartBytes); err != nil {
			return fmt.Errorf("invalid values.yaml: %w", err)
		}
		chartValues = string(chartBytes)
	}

	// Parse parameter templates from CLI flags
	parameterTemplates, err := parseParameterTemplates(cmd)
	if err != nil {
		return err
	}

	gresp, err := catalogClient.CatalogServiceGetApplicationWithResponse(ctx, projectName, name, version,
		auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
		return err
	}

	application := gresp.JSON200.Application

	// Create profile with chart values only if provided
	newProfile := catapi.CatalogV3Profile{
		Name:               profileName,
		DisplayName:        &displayName,
		Description:        &description,
		ParameterTemplates: parameterTemplates,
	}
	if chartValues != "" {
		newProfile.ChartValues = &chartValues
	}

	profiles := append(*application.Profiles, newProfile)

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
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error creating profile %s of application %s:%s",
		profileName, name, version)); err != nil {
		return err
	}
	fmt.Printf("Profile '%s' created successfully for application '%s:%s'\n", profileName, name, version)
	return nil
}

func runListProfilesCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
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
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
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
			printProfiles(writer, &[]catapi.CatalogV3Profile{profile}, verbose)
			return writer.Flush()
		}
	}
	return errors.NewNotFound("profile %s for application %s:%s not found", profileName, name, version)
}

func runSetProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
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
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
		return err
	}

	// Scan through the profiles and update the named profile
	application := gresp.JSON200.Application
	profiles := *application.Profiles

	var profile *catapi.CatalogV3Profile
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

	if profile == nil {
		return errors.NewNotFound("profile %s for application %s:%s not found", profileName, name, version)
	}

	profile.DisplayName = getFlagOrDefault(cmd, "display-name", profile.DisplayName)
	profile.Description = getFlagOrDefault(cmd, "description", profile.Description)

	// If parameter templates were specified, update them
	parameterTemplates, err := parseParameterTemplates(cmd)
	if err != nil {
		return err
	}
	if len(*parameterTemplates) > 0 {
		profile.ParameterTemplates = parameterTemplates
	}

	// If the chart-values flag was given, fetch the new content to replace the existing one
	newChartValuesPath := *getFlag(cmd, "chart-values")
	if len(newChartValuesPath) > 0 {
		chartValueBytes, err := readInputWithLimit(*getFlag(cmd, "chart-values"))
		if err != nil {
			return fmt.Errorf("error reading chart-values content: %w", err)
		}
		if err := validateValuesYAML(chartValueBytes); err != nil {
			return fmt.Errorf("invalid values.yaml: %w", err)
		}
		newChartValues := string(chartValueBytes)
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
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error updating profile %s of application %s:%s",
		profileName, name, version)); err != nil {
		return err
	}
	fmt.Printf("Profile '%s' updated successfully for application '%s:%s'\n", profileName, name, version)
	return nil
}

func runDeleteProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
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
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("application %s:%s not found", name, version)); err != nil {
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
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting profile %s of application %s:%s",
		profileName, name, version)); err != nil {
		return err
	}
	fmt.Printf("Profile '%s' deleted successfully from application '%s:%s'\n", profileName, name, version)
	return nil
}

func validateValuesYAML(data []byte) error {
	var out interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	// Optionally enforce top-level map/object
	if _, ok := out.(map[interface{}]interface{}); !ok {
		return fmt.Errorf("values.yaml must have a map/object at the top level")
	}
	return nil
}
