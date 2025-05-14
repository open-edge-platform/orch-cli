// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var OSProfileHeader = fmt.Sprintf("%s\t%s\t%s", "Name", "Architecture", "Security Feature")
var OSProfileHeaderGet = fmt.Sprintf("%s\t%s", "OS Profile Field", "Value")

type OSProfileSpec struct {
	Name            string `yaml:"name"`
	Type            string `yaml:"type"`
	Provider        string `yaml:"provider"`
	Architecture    string `yaml:"architecture"`
	ProfileName     string `yaml:"profileName"`
	OsImageUrl      string `yaml:"osImageUrl"`
	OsImageSha256   string `yaml:"osImageSha256"`
	OsImageVersion  string `yaml:"osImageVersion"`
	OSPackageURL    string `yaml:"osPackageManifestURL"`
	SecurityFeature string `yaml:"securityFeature"`
	PlatformBundle  string `yaml:"platformBundle"`
}

type NestedSpec struct {
	AppVersion string        `yaml:"appVersion"`
	Spec       OSProfileSpec `yaml:"spec"`
}

// Prints OS Profiles in tabular format
func printOSProfiles(writer io.Writer, OSProfiles *[]infra.OperatingSystemResource, verbose bool) {
	for _, osp := range *OSProfiles {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *osp.Name, *osp.Architecture, *osp.SecurityFeature)
		} else {
			_, _ = fmt.Fprintf(writer, "Name:\t %s\n", *osp.Name)
			_, _ = fmt.Fprintf(writer, "Profile Name:\t %s\n", *osp.ProfileName)
			_, _ = fmt.Fprintf(writer, "Security Feature:\t %v\n", toJSON(osp.SecurityFeature))
			_, _ = fmt.Fprintf(writer, "Architecture:\t %s\n", *osp.Architecture)
			_, _ = fmt.Fprintf(writer, "Repository URL:\t %s\n", *osp.RepoUrl)
			_, _ = fmt.Fprintf(writer, "sha256:\t %v\n", osp.Sha256)
			_, _ = fmt.Fprintf(writer, "Kernel Command:\t %v\n\n", toJSON(osp.KernelCommand))
		}
	}
}

// Prints output details of OS Profiles
func printOSProfile(writer io.Writer, OSProfile *infra.OperatingSystemResource) {

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *OSProfile.Name)
	_, _ = fmt.Fprintf(writer, "Profile Name: \t%s\n", *OSProfile.ProfileName)
	_, _ = fmt.Fprintf(writer, "OS Resource ID: \t%s\n", *OSProfile.OsResourceID)
	_, _ = fmt.Fprintf(writer, "version: \t%v\n", toJSON(OSProfile.ProfileVersion))
	_, _ = fmt.Fprintf(writer, "sha256: \t%v\n", OSProfile.Sha256)
	_, _ = fmt.Fprintf(writer, "Image ID: \t%s\n", *OSProfile.ImageId)
	_, _ = fmt.Fprintf(writer, "Image URL: \t%s\n", *OSProfile.ImageUrl)
	_, _ = fmt.Fprintf(writer, "Repository URL: \t%s\n", *OSProfile.RepoUrl)
	_, _ = fmt.Fprintf(writer, "Security Feature: \t%v\n", toJSON(OSProfile.SecurityFeature))
	_, _ = fmt.Fprintf(writer, "Architecture: \t%s\n", *OSProfile.Architecture)
	_, _ = fmt.Fprintf(writer, "OS type: \t%s\n", *OSProfile.OsType)
	_, _ = fmt.Fprintf(writer, "OS provider: \t%s\n", *OSProfile.OsProvider)
	_, _ = fmt.Fprintf(writer, "Platform Bundle: \t%s\n", *OSProfile.PlatformBundle)
	_, _ = fmt.Fprintf(writer, "Update Sources: \t%v\n", OSProfile.UpdateSources)
	_, _ = fmt.Fprintf(writer, "Installed Packages: \t%v\n", toJSON(OSProfile.InstalledPackages))
	_, _ = fmt.Fprintf(writer, "Created: \t%v\n", OSProfile.Timestamps.CreatedAt)
	_, _ = fmt.Fprintf(writer, "Updated: \t%v\n", OSProfile.Timestamps.UpdatedAt)

}

// Helper function to verify that the input file exists and is of right format
func verifyOSProfileInput(path string) error {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" {
		return errors.New("os Profile input must be a yaml file")
	}

	return nil
}

// Helper function to unmarshal yaml file
func readOSProfileFromYaml(path string) (*NestedSpec, error) {

	var input NestedSpec
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &input)
	if err != nil {
		log.Fatalf("error unmarshalling YAML: %v", err)
	}

	return &input, nil
}

// Filters list of profiles to find one with specific name
func filterProfilesByName(OSProfiles *[]infra.OperatingSystemResource, name string) (*infra.OperatingSystemResource, error) {
	for _, profile := range *OSProfiles {
		if *profile.Name == name {
			return &profile, nil
		}
	}
	return nil, errors.New("no os profile matches the given name")
}

func getGetOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osprofile <name> [flags]",
		Short: "Get an OS profile",
		Args:  cobra.ExactArgs(1),
		RunE:  runGetOSProfileCommand,
	}
	return cmd
}

func getListOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osprofile <name> [flags]",
		Short: "List OS profiles",
		RunE:  runListOSProfileCommand,
	}
	return cmd
}

func getCreateOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osprofile <name> [flags]",
		Short: "List OS profiles",
		Args:  cobra.ExactArgs(1),
		RunE:  runCreateOSProfileCommand,
	}
	return cmd
}

func getDeleteOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osprofile <name> [flags]",
		Short: "Delete an OS profile",
		Args:  cobra.ExactArgs(1),
		RunE:  runDeleteOSProfileCommand,
	}
	return cmd
}

// Gets specific OS Profile - retrieves list of profiles and then filters and outputs
// specifc profile by name
func runGetOSProfileCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := OSProfileClient.GetV1ProjectsProjectNameComputeOsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameComputeOsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OSProfileHeaderGet, "error getting OS Profile"); !proceed {
		return err
	}

	name := args[0]
	profile, err := filterProfilesByName(resp.JSON200.OperatingSystemResources, name)
	if err != nil {
		return err
	}

	printOSProfile(writer, profile)
	return writer.Flush()
}

// Lists all OS Profiles - retrieves all profiles and displays selected information in tabular format
func runListOSProfileCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := OSProfileClient.GetV1ProjectsProjectNameComputeOsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameComputeOsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OSProfileHeader, "error getting OS Profiles"); !proceed {
		return err
	}

	printOSProfiles(writer, resp.JSON200.OperatingSystemResources, verbose)

	return writer.Flush()
}

// Creates OS Profile - checks if a profile already exists and the creates it if it does not using the input .yaml file
func runCreateOSProfileCommand(cmd *cobra.Command, args []string) error {
	path := args[0]

	err := verifyOSProfileInput(path)
	if err != nil {
		return err
	}

	spec, err := readOSProfileFromYaml(path)
	if err != nil {
		return err
	}

	ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	//TODO Delete name check once API accepts only unique names
	gresp, err := OSProfileClient.GetV1ProjectsProjectNameComputeOsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameComputeOsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err = checkResponse(gresp.HTTPResponse, "Error getting OS profiles"); err != nil {
		return err
	}

	_, err = filterProfilesByName(gresp.JSON200.OperatingSystemResources, spec.Spec.Name)
	if err == nil {
		return fmt.Errorf("OS Profile %s already exists", spec.Spec.Name)
	}
	// End TODO

	resp, err := OSProfileClient.PostV1ProjectsProjectNameComputeOsWithResponse(ctx, projectName,
		infra.PostV1ProjectsProjectNameComputeOsJSONRequestBody{
			Name:            &spec.Spec.Name,
			Architecture:    &spec.Spec.Architecture,
			ImageUrl:        &spec.Spec.OsImageUrl,
			ImageId:         &spec.Spec.OsImageVersion,
			OsType:          (*infra.OperatingSystemType)(&spec.Spec.Type),
			OsProvider:      (*infra.OperatingSystemProvider)(&spec.Spec.Provider),
			ProfileName:     &spec.Spec.ProfileName,
			RepoUrl:         &spec.Spec.OsImageUrl,
			SecurityFeature: (*infra.SecurityFeature)(&spec.Spec.SecurityFeature),
			Sha256:          spec.Spec.OsImageSha256,
			UpdateSources:   []string{""},
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating OS Profile from %s", path))
}

// Deletes OS Profile - checks if a profile already exists and then deletes it if it does
func runDeleteOSProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	gresp, err := OSProfileClient.GetV1ProjectsProjectNameComputeOsWithResponse(ctx, projectName,
		&infra.GetV1ProjectsProjectNameComputeOsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err = checkResponse(gresp.HTTPResponse, "Error getting OS profiles"); err != nil {
		return err
	}

	name := args[0]
	profile, err := filterProfilesByName(gresp.JSON200.OperatingSystemResources, name)
	if err != nil {
		return err
	}

	resp, err := OSProfileClient.DeleteV1ProjectsProjectNameComputeOsOSResourceIDWithResponse(ctx, projectName,
		*profile.OsResourceID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting OS profile %s", name))

}
