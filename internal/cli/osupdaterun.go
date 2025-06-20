// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
)

const listOSUpdateRunExamples = `# List all OS Update Policies
orch-cli list osupdaterun --project some-project
`

const getOSUpdateRunExamples = `# Get detailed information about specific OS Update Run using the run name
orch-cli get osupdaterun runname --project some-project`

const deleteOSUpdateRunExamples = `#Delete an OS Update Run  using it's name
orch-cli delete osupdaterun run --project some-project`

var OSUpdateRunHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Value", "Value")
var OSUpdateRunGet = fmt.Sprintf("\n%s\t%s", "OS Run", "Value")

// Prints OS Profiles in tabular format
func printOSUpdateRuns(writer io.Writer, OSUpdateRuns []infra.OSUpdateRun, verbose bool) {
	for _, run := range OSUpdateRuns {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *run.Name, *run.ResourceId, *run.Status)
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
func printOSUpdateRun(writer io.Writer, OSUpdateRun *infra.OSUpdateRun) {

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

func getGetOSUpdateRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdaterun <name> [flags]",
		Short:   "Get an OS Update run",
		Example: getOSUpdateRunExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runGetOSUpdateRunCommand,
	}
	return cmd
}

func getListOSUpdateRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdaterun [flags]",
		Short:   "List all OS Update policies",
		Example: listOSUpdateRunExamples,
		RunE:    runListOSUpdateRunCommand,
	}
	return cmd
}

func getDeleteOSUpdateRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdaterun <name> [flags]",
		Short:   "Delete an OS Update run",
		Example: deleteOSUpdateRunExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runDeleteOSUpdateRunCommand,
	}
	return cmd
}

// Gets specific OSUpdateRun - retrieves list of policies and then filters and outputs
// specifc run by name
func runGetOSUpdateRunCommand(cmd *cobra.Command, args []string) error {
	uprun := args[0]

	writer, verbose := getOutputContext(cmd)
	ctx, OSUpdateRunClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := OSUpdateRunClient.OSUpdateRunGetOSUpdateRunWithResponse(ctx, projectName,
		uprun, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OSProfileHeaderGet, "error getting OS Update run"); !proceed {
		return err
	}

	printOSUpdateRun(writer, resp.JSON200)
	return writer.Flush()
}

// Lists all OS Update policies - retrieves all policies and displays selected information in tabular format
func runListOSUpdateRunCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	filtflag, _ := cmd.Flags().GetString("filter")
	filter := filterHelper(filtflag)

	ctx, OSUpdateRunClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	//TODO handle multiple pages
	resp, err := OSUpdateRunClient.OSUpdateRunListOSUpdateRunWithResponse(ctx, projectName,
		&infra.OSUpdateRunListOSUpdateRunParams{
			Filter: filter,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OSUpdateRunHeader, "error getting OS Update Runs"); !proceed {
		return err
	}

	printOSUpdateRuns(writer, resp.JSON200.OsUpdateRuns, verbose)

	// return writer.Flush()
	return nil
}

// Deletes OS Update Run - checks if a run  already exists and then deletes it if it does
func runDeleteOSUpdateRunCommand(cmd *cobra.Command, args []string) error {
	osrun := args[0]

	ctx, OSUpdateRunClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := OSUpdateRunClient.OSUpdateRunDeleteOSUpdateRunWithResponse(ctx, projectName,
		osrun, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting OS Update run %s", osrun))
}
