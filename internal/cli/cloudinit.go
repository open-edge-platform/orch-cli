// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listCloudInitExamples = `# List all Cloud Init resources
orch-cli list cloudinit --project some-project
`

const getCloudInitExamples = `# Get detailed information about specific Cloud Init resource
orch-cli get cloudinit policyname --project some-project`

const createCloudInitExamples = `# Create a Cloud Init resource
orch-cli create cloudinit   --project some-project`

const deleteCloudInitExamples = `#Delete a Cloud Init resource
orch-cli delete cloudinit name --project some-project`

var CloudInitHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Value", "Value")
var CloudInitGet = fmt.Sprintf("\n%s\t%s", "Cloud Init", "Value")

// Prints OS Profiles in tabular format
func printCloudInits(writer io.Writer, CloudInit []infra.CustomConfigResource, verbose bool) {
	for _, cinit := range CloudInit {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", cinit.Name, *cinit.ResourceId, *cinit.Description)
		} else {
			// _, _ = fmt.Fprintf(writer, "\nName:\t %s\n", *osp.Name)
			// _, _ = fmt.Fprintf(writer, "Profile Name:\t %s\n", *osp.ProfileName)
			// _, _ = fmt.Fprintf(writer, "Security Feature:\t %v\n", toJSON(osp.SecurityFeature))
			// _, _ = fmt.Fprintf(writer, "Architecture:\t %s\n", *osp.Architecture)
			// _, _ = fmt.Fprintf(writer, "Repository URL:\t %s\n", *osp.RepoUrl)
			// _, _ = fmt.Fprintf(writer, "sha256:\t %v\n", osp.Sha256)
			// _, _ = fmt.Fprintf(writer, "Kernel Command:\t %v\n", toJSON(osp.KernelCommand))
		}
	}
}

// Prints output details of OS Profiles
func printCloudInit(writer io.Writer, CloudInit *infra.CustomConfigResource) {

	// _, _ = fmt.Fprintf(writer, "Name: \t%s\n", *OSProfile.Name)
	// _, _ = fmt.Fprintf(writer, "Profile Name: \t%s\n", *OSProfile.ProfileName)
	// _, _ = fmt.Fprintf(writer, "OS Resource ID: \t%s\n", *OSProfile.OsResourceID)
	// _, _ = fmt.Fprintf(writer, "version: \t%v\n", toJSON(OSProfile.ProfileVersion))
	// _, _ = fmt.Fprintf(writer, "sha256: \t%v\n", OSProfile.Sha256)
	// _, _ = fmt.Fprintf(writer, "Image ID: \t%s\n", *OSProfile.ImageId)
	// _, _ = fmt.Fprintf(writer, "Image URL: \t%s\n", *OSProfile.ImageUrl)
	// _, _ = fmt.Fprintf(writer, "Repository URL: \t%s\n", *OSProfile.RepoUrl)
	// _, _ = fmt.Fprintf(writer, "Security Feature: \t%v\n", toJSON(OSProfile.SecurityFeature))
	// _, _ = fmt.Fprintf(writer, "Architecture: \t%s\n", *OSProfile.Architecture)
	// _, _ = fmt.Fprintf(writer, "OS type: \t%s\n", *OSProfile.OsType)
	// _, _ = fmt.Fprintf(writer, "OS provider: \t%s\n", *OSProfile.OsProvider)
	// _, _ = fmt.Fprintf(writer, "Platform Bundle: \t%s\n", *OSProfile.PlatformBundle)
	// _, _ = fmt.Fprintf(writer, "Update Sources: \t%v\n", OSProfile.UpdateSources)
	// _, _ = fmt.Fprintf(writer, "Installed Packages: \t%v\n", toJSON(OSProfile.InstalledPackages))
	// _, _ = fmt.Fprintf(writer, "Created: \t%v\n", OSProfile.Timestamps.CreatedAt)
	// _, _ = fmt.Fprintf(writer, "Updated: \t%v\n", OSProfile.Timestamps.UpdatedAt)

}

// // Helper function to verify that the input file exists and is of right format
// func verifyOSProfileInput(path string) error {

// 	if _, err := os.Stat(path); os.IsNotExist(err) {
// 		return fmt.Errorf("file does not exist: %s", path)
// 	}

// 	ext := strings.ToLower(filepath.Ext(path))
// 	if ext != ".yaml" && ext != ".yml" {
// 		return errors.New("os Profile input must be a yaml file")
// 	}

// 	return nil
// }

// // Helper function to unmarshal yaml file
// func readOSProfileFromYaml(path string) (*NestedSpec, error) {

// 	var input NestedSpec
// 	data, err := os.ReadFile(path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = yaml.Unmarshal(data, &input)
// 	if err != nil {
// 		log.Fatalf("error unmarshalling YAML: %v", err)
// 	}

// 	return &input, nil
// }

// // Filters list of profiles to find one with specific name
// func filterProfilesByName(OSProfiles []infra.OperatingSystemResource, name string) (*infra.OperatingSystemResource, error) {
// 	for _, profile := range OSProfiles {
// 		if *profile.Name == name {
// 			return &profile, nil
// 		}
// 	}
// 	return nil, errors.New("no os profile matches the given name")
// }

func getGetCloudInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cloudinit <name> [flags]",
		Short:   "Get a Cloud Init configuration",
		Example: getCloudInitExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runGetCloudInitCommand,
	}
	return cmd
}

func getListCloudInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cloudinit [flags]",
		Short:   "List all Cloud Init configurations",
		Example: listCloudInitExamples,
		RunE:    runListCloudInitCommand,
	}
	cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of cloud init list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter <filter> see https://google.aip.dev/160 and API spec.")
	return cmd
}

func getCreateCloudInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cloudinit  [flags]",
		Short:   "Creates Cloud Init configuration",
		Example: createCloudInitExamples,
		Args:    cobra.ExactArgs(2),
		RunE:    runCreateCloudInitCommand,
	}
	return cmd
}

func getDeleteCloudInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cloudinit <name> [flags]",
		Short:   "Delete a Cloud Init config",
		Example: deleteCloudInitExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runDeleteCloudInitCommand,
	}
	return cmd
}

// Gets specific Cloud Init configuration bu resource ID
func runGetCloudInitCommand(cmd *cobra.Command, args []string) error {
	CIID := args[0]

	writer, verbose := getOutputContext(cmd)
	ctx, cloudInitClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := cloudInitClient.CustomConfigServiceGetCustomConfigWithResponse(ctx, projectName,
		CIID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		CloudInitGet, "error getting Cloud Init configuration"); !proceed {
		return err
	}

	printCloudInit(writer, resp.JSON200)
	return writer.Flush()
}

// Lists all Cloud Init configrations - retrieves all configurations and displays selected information in tabular format
func runListCloudInitCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	filtflag, _ := cmd.Flags().GetString("filter")
	filter := filterHelper(filtflag)

	ctx, cloudInitClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := cloudInitClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
		&infra.CustomConfigServiceListCustomConfigsParams{
			Filter: filter,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		CloudInitHeader, "error getting Cloud Init configurations"); !proceed {
		return err
	}

	printCloudInits(writer, resp.JSON200.CustomConfigs, verbose)

	return writer.Flush()
}

// Creates Cloud Init config
func runCreateCloudInitCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	path := args[1]

	// err := verifyOSProfileInput(path)
	// if err != nil {
	// 	return err
	// }

	// spec, err := readOSProfileFromYaml(path)
	// if err != nil {
	// 	return err
	// }

	//TODO remove hardcoded and read from yaml
	config := "config"
	desc := "A description"

	ctx, cloudInitClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := cloudInitClient.CustomConfigServiceCreateCustomConfigWithResponse(ctx, projectName,
		infra.CustomConfigServiceCreateCustomConfigJSONRequestBody{
			Name:        name,
			Description: &desc,
			Config:      config,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating Cloud Init config from %s", path))
}

// Deletes OS Update Policy - checks if a policy  already exists and then deletes it if it does
func runDeleteCloudInitCommand(cmd *cobra.Command, args []string) error {

	policy := args[0]
	ctx, cloudInitClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := cloudInitClient.CustomConfigServiceDeleteCustomConfigWithResponse(ctx, projectName,
		policy, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting Cloud Init config %s", policy))
}
