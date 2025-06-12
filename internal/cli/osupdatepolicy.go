// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
)

const listOSUpdatePolicyExamples = `# List all OS Update Policies
orch-cli list osupdatepolicy --project some-project
`

const getOSUpdatePolicyExamples = `# Get detailed information about specific OS Update Policy using the policy name
orch-cli get osupdatepolicy policyname --project some-project`

const createOSUpdatePolicyExamples = `# Create an OS Update Policy.
orch-cli create osupdatepolicy   --project some-project`

const deleteOSUpdatePolicyExamples = `#Delete an OS Update Policy  using it's name
orch-cli delete osupdatepolicy policy --project some-project`

var OSUpdatePolicyHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Value", "Value")
var OSUpdatePolicyrGet = fmt.Sprintf("\n%s\t%s", "OS Policy", "Value")

// Prints OS Profiles in tabular format
func printOSUpdatePolicies(writer io.Writer, OSUpdatePolicies []infra.OSUpdatePolicy, verbose bool) {
	// for _, osp := range OSProfiles {
	// 	if !verbose {
	// 		fmt.Fprintf(writer, "%s\t%s\t%s\n", *osp.Name, *osp.Architecture, *osp.SecurityFeature)
	// 	} else {
	// 		_, _ = fmt.Fprintf(writer, "\nName:\t %s\n", *osp.Name)
	// 		_, _ = fmt.Fprintf(writer, "Profile Name:\t %s\n", *osp.ProfileName)
	// 		_, _ = fmt.Fprintf(writer, "Security Feature:\t %v\n", toJSON(osp.SecurityFeature))
	// 		_, _ = fmt.Fprintf(writer, "Architecture:\t %s\n", *osp.Architecture)
	// 		_, _ = fmt.Fprintf(writer, "Repository URL:\t %s\n", *osp.RepoUrl)
	// 		_, _ = fmt.Fprintf(writer, "sha256:\t %v\n", osp.Sha256)
	// 		_, _ = fmt.Fprintf(writer, "Kernel Command:\t %v\n", toJSON(osp.KernelCommand))
	// 	}
	// }
}

// Prints output details of OS Profiles
func printOSUpdatePolicy(writer io.Writer, OSUpdatePolicy *infra.OSUpdatePolicy) {

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

func getGetOSUpdatePolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdatepolicy <name> [flags]",
		Short:   "Get an OS Update policy",
		Example: getOSUpdatePolicyExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runGetOSUpdatePolicyCommand,
	}
	return cmd
}

func getListOSUpdatePolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdatepolicy [flags]",
		Short:   "List all OS Update policies",
		Example: listOSUpdatePolicyExamples,
		RunE:    runListOSUpdatePolicyCommand,
	}
	//cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of host list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter \"osType=OS_TYPE_IMMUTABLE\" see https://google.aip.dev/160 and API spec.")
	return cmd
}

func getCreateOSUpdatePolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdatepolicy  [flags]",
		Short:   "Creates OS Update policy",
		Example: createOSUpdatePolicyExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runCreateOSUpdatePolicyCommand,
	}
	return cmd
}

func getDeleteOSUpdatePolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdatepolicy <name> [flags]",
		Short:   "Delete an OS Update policy",
		Example: deleteOSUpdatePolicyExamples,
		Args:    cobra.ExactArgs(1),
		RunE:    runDeleteOSUpdatePolicyCommand,
	}
	return cmd
}

// Gets specific OSUpdatePolicy - retrieves list of policies and then filters and outputs
// specifc policy by name
func runGetOSUpdatePolicyCommand(cmd *cobra.Command, args []string) error {
	// writer, verbose := getOutputContext(cmd)
	// ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	// if err != nil {
	// 	return err
	// }

	// resp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
	// 	&infra.OperatingSystemServiceListOperatingSystemsParams{}, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
	// 	OSProfileHeaderGet, "error getting OS Profile"); !proceed {
	// 	return err
	// }

	// name := args[0]
	// profile, err := filterProfilesByName(resp.JSON200.OperatingSystemResources, name)
	// if err != nil {
	// 	return err
	// }

	// printOSProfile(writer, profile)
	// return writer.Flush()
	return nil
}

// Lists all OS Update policies - retrieves all policies and displays selected information in tabular format
func runListOSUpdatePolicyCommand(cmd *cobra.Command, _ []string) error {
	// writer, verbose := getOutputContext(cmd)

	// filtflag, _ := cmd.Flags().GetString("filter")
	// filter := filterHelper(filtflag)

	// ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	// if err != nil {
	// 	return err
	// }

	// resp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
	// 	&infra.OperatingSystemServiceListOperatingSystemsParams{
	// 		Filter: filter,
	// 	}, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
	// 	OSProfileHeader, "error getting OS Profiles"); !proceed {
	// 	return err
	// }

	// printOSProfiles(writer, resp.JSON200.OperatingSystemResources, verbose)

	// return writer.Flush()
	return nil
}

// Creates OS Update Policy - checks if a OS Update Policy already exists and then creates it if it does not
func runCreateOSUpdatePolicyCommand(cmd *cobra.Command, args []string) error {
	// path := args[0]

	// err := verifyOSProfileInput(path)
	// if err != nil {
	// 	return err
	// }

	// spec, err := readOSProfileFromYaml(path)
	// if err != nil {
	// 	return err
	// }

	// ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	// if err != nil {
	// 	return err
	// }

	// //TODO Delete name check once API accepts only unique names
	// gresp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
	// 	&infra.OperatingSystemServiceListOperatingSystemsParams{}, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// if err = checkResponse(gresp.HTTPResponse, "Error getting OS profiles"); err != nil {
	// 	return err
	// }

	// _, err = filterProfilesByName(gresp.JSON200.OperatingSystemResources, spec.Spec.Name)
	// if err == nil {
	// 	return fmt.Errorf("OS Profile %s already exists", spec.Spec.Name)
	// }
	// // End TODO

	// resp, err := OSProfileClient.OperatingSystemServiceCreateOperatingSystemWithResponse(ctx, projectName,
	// 	infra.OperatingSystemServiceCreateOperatingSystemJSONRequestBody{
	// 		Name:            &spec.Spec.Name,
	// 		Architecture:    &spec.Spec.Architecture,
	// 		ImageUrl:        &spec.Spec.OsImageURL,
	// 		ImageId:         &spec.Spec.OsImageVersion,
	// 		OsType:          (*infra.OsType)(&spec.Spec.Type),
	// 		OsProvider:      (*infra.OsProviderKind)(&spec.Spec.Provider),
	// 		ProfileName:     &spec.Spec.ProfileName,
	// 		RepoUrl:         &spec.Spec.OsImageURL,
	// 		SecurityFeature: (*infra.SecurityFeature)(&spec.Spec.SecurityFeature),
	// 		Sha256:          spec.Spec.OsImageSha256,
	// 	}, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }
	// return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating OS Profile from %s", path))
	return nil
}

// Deletes OS Update Policy - checks if a policy  already exists and then deletes it if it does
func runDeleteOSUpdatePolicyCommand(cmd *cobra.Command, args []string) error {
	// ctx, OSProfileClient, projectName, err := getInfraServiceContext(cmd)
	// if err != nil {
	// 	return err
	// }

	// gresp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
	// 	&infra.OperatingSystemServiceListOperatingSystemsParams{}, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// if err = checkResponse(gresp.HTTPResponse, "Error getting OS profiles"); err != nil {
	// 	return err
	// }

	// name := args[0]
	// profile, err := filterProfilesByName(gresp.JSON200.OperatingSystemResources, name)
	// if err != nil {
	// 	return err
	// }

	// resp, err := OSProfileClient.OperatingSystemServiceDeleteOperatingSystemWithResponse(ctx, projectName,
	// 	*profile.OsResourceID, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting OS profile %s", name))
	return nil
}
