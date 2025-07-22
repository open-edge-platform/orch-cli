// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
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
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const listOSUpdatePolicyExamples = `# List all OS Update Policies
orch-cli list osupdatepolicy --project some-project
`

const getOSUpdatePolicyExamples = `# Get detailed information about specific OS Update Policy using the policy name
orch-cli get osupdatepolicy <resourceID> --project some-project`

const createOSUpdatePolicyExamples = `# Create an OS Update Policy.
orch-cli create osupdatepolicy path/to/osupdatepolicy.yaml  --project some-project

Sample OS update policy file format for immutable OS
appVersion: apps/v1
spec:
  name: myupdate
  description: "an update profile"
  updatePolicy: "UPDATE_POLICY_LATEST"
`

const deleteOSUpdatePolicyExamples = `#Delete an OS Update Policy  using it's name
orch-cli delete <resourceID> policy --project some-project`

var OSUpdatePolicyHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Resource ID", "Description")

type OSUpdatePolicy struct {
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	InstallPackages string   `yaml:"installPackages"`
	KernelCommand   string   `yaml:"kernelCommand"`
	TargetOS        string   `yaml:"targetOs"`
	UpdateSources   []string `yaml:"updateSources"`
	UpdatePolicy    string   `yaml:"updatePolicy"`
}

type UpdateNestedSpec struct {
	Spec OSUpdatePolicy `yaml:"spec"`
}

// Filters list of profiles to find one with specific name
func filterPoliciesByName(OSPolicies []infra.OSUpdatePolicy, name string) (*infra.OSUpdatePolicy, error) {
	for _, policy := range OSPolicies {
		if policy.Name == name {
			return &policy, nil
		}
	}
	return nil, errors.New("no os update policy matches the given name")
}

// Prints OS Profiles in tabular format
func printOSUpdatePolicies(writer io.Writer, OSUpdatePolicies []infra.OSUpdatePolicy, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\t%s\t%s\n", "Name", "Resource ID", "Target OS ID", "Description", "Created", "Updated")
	}
	for _, osup := range OSUpdatePolicies {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", osup.Name, *osup.ResourceId, *osup.Description)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", osup.Name, *osup.ResourceId, *osup.TargetOsId, *osup.Description, *osup.Timestamps.CreatedAt, *osup.Timestamps.UpdatedAt)
		}
	}
}

// Prints output details of OS Profiles
func printOSUpdatePolicy(writer io.Writer, OSUpdatePolicy *infra.OSUpdatePolicy) {

	_, _ = fmt.Fprintf(writer, "Name:\t %s\n", OSUpdatePolicy.Name)
	_, _ = fmt.Fprintf(writer, "Resource ID:\t %s\n", *OSUpdatePolicy.ResourceId)
	_, _ = fmt.Fprintf(writer, "Target OS ID:\t %s\n", *OSUpdatePolicy.TargetOsId)
	_, _ = fmt.Fprintf(writer, "Description:\t %v\n", *OSUpdatePolicy.Description)
	_, _ = fmt.Fprintf(writer, "Install Packages:\t %s\n", *OSUpdatePolicy.InstallPackages)
	_, _ = fmt.Fprintf(writer, "Update Policy:\t %s\n", *OSUpdatePolicy.UpdatePolicy)
	_, _ = fmt.Fprintf(writer, "Create at:\t %v\n", *OSUpdatePolicy.Timestamps.CreatedAt)
	_, _ = fmt.Fprintf(writer, "Updated at:\t %v\n", *OSUpdatePolicy.Timestamps.CreatedAt)

}

// Helper function to verify that the input file exists and is of right format
func verifyUpdateProfileInput(path string) error {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" {
		return errors.New("update profile input must be a yaml file")
	}

	return nil
}

// Helper function to unmarshal yaml file
func readUpdateProfileFromYaml(path string) (*UpdateNestedSpec, error) {

	var input UpdateNestedSpec
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
	cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of host list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter \"osType=OS_TYPE_IMMUTABLE\" see https://google.aip.dev/160 and API spec.")
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

	writer, verbose := getOutputContext(cmd)
	ctx, OSUpdatePolicyClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	// In future get policy by name istead of resourceid
	// name := args[0]

	// lresp, err := OSUpdatePolicyClient.OSUpdatePolicyListOSUpdatePolicyWithResponse(ctx, projectName,
	// 	&infra.OSUpdatePolicyListOSUpdatePolicyParams{}, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// if err = checkResponse(lresp.HTTPResponse, "Error getting OS Update policies"); err != nil {
	// 	return err
	// }

	// policy, err := filterPoliciesByName(lresp.JSON200.OsUpdatePolicies, name)
	// if err != nil {
	// 	return err
	// }

	// policyID := *policy.ResourceId

	policyID := args[0]

	resp, err := OSUpdatePolicyClient.OSUpdatePolicyGetOSUpdatePolicyWithResponse(ctx, projectName,
		policyID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting OS Update Policy"); !proceed {
		return err
	}

	printOSUpdatePolicy(writer, resp.JSON200)
	return writer.Flush()
}

// Lists all OS Update policies - retrieves all policies and displays selected information in tabular format
func runListOSUpdatePolicyCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	filtflag, _ := cmd.Flags().GetString("filter")
	filter := filterHelper(filtflag)

	ctx, OSUPolicyClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	resp, err := OSUPolicyClient.OSUpdatePolicyListOSUpdatePolicyWithResponse(ctx, projectName,
		&infra.OSUpdatePolicyListOSUpdatePolicyParams{
			Filter: filter,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OSUpdatePolicyHeader, "error getting OS Update Policies"); !proceed {
		return err
	}

	printOSUpdatePolicies(writer, resp.JSON200.OsUpdatePolicies, verbose)

	return writer.Flush()
}

// Creates OS Update Policy - checks if a OS Update Policy already exists and then creates it if it does not
func runCreateOSUpdatePolicyCommand(cmd *cobra.Command, args []string) error {
	path := args[0]
	writer, verbose := getOutputContext(cmd)

	err := verifyUpdateProfileInput(path)
	if err != nil {
		return err
	}

	spec, err := readUpdateProfileFromYaml(path)
	if err != nil {
		return err
	}

	ctx, OSUPolicyClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	var profile *infra.OperatingSystemResource
	var updpol infra.UpdatePolicy
	var packages *string
	var kernel *string
	var sources *[]string
	if spec.Spec.TargetOS != "" {
		//check if target OS exists
		oresp, err := OSUPolicyClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
			&infra.OperatingSystemServiceListOperatingSystemsParams{}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		if proceed, err := processResponse(oresp.HTTPResponse, oresp.Body, writer, verbose,
			OSProfileHeaderGet, "error getting OS Profile"); !proceed {
			return err
		}

		name := spec.Spec.TargetOS
		profile, err = filterProfilesByName(oresp.JSON200.OperatingSystemResources, name)
		if err != nil {
			return err
		}
	}
	if spec.Spec.UpdatePolicy != "" {
		updpol = infra.UpdatePolicy(spec.Spec.UpdatePolicy)
	}

	if spec.Spec.InstallPackages != "" {
		packages = &spec.Spec.InstallPackages
	}

	if spec.Spec.KernelCommand != "" {
		kernel = &spec.Spec.KernelCommand
	}

	if spec.Spec.UpdateSources != nil {
		sources = &spec.Spec.UpdateSources
	}

	//Create policy
	resp, err := OSUPolicyClient.OSUpdatePolicyCreateOSUpdatePolicyWithResponse(ctx, projectName,
		infra.OSUpdatePolicyCreateOSUpdatePolicyJSONRequestBody{
			Name:            spec.Spec.Name,
			Description:     &spec.Spec.Description,
			InstallPackages: packages,
			KernelCommand:   kernel,
			TargetOs:        profile,
			UpdateSources:   sources,
			UpdatePolicy:    &updpol,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating OS Update Profiles from %s", path))
}

// Deletes OS Update Policy - checks if a policy  already exists and then deletes it if it does
func runDeleteOSUpdatePolicyCommand(cmd *cobra.Command, args []string) error {

	ctx, OSUPolicyClient, projectName, err := getInfraServiceContext(cmd)
	if err != nil {
		return err
	}

	// In future delete by name instead of resource id
	// name := args[0]
	// lresp, err := OSUPolicyClient.OSUpdatePolicyListOSUpdatePolicyWithResponse(ctx, projectName,
	// 	&infra.OSUpdatePolicyListOSUpdatePolicyParams{}, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	// if err = checkResponse(lresp.HTTPResponse, "Error getting OS Update policies"); err != nil {
	// 	return err
	// }

	// policy, err := filterPoliciesByName(lresp.JSON200.OsUpdatePolicies, name)
	// if err != nil {
	// 	return err
	// }

	// policyID := *policy.ResourceId

	policyID := args[0]

	resp, err := OSUPolicyClient.OSUpdatePolicyDeleteOSUpdatePolicyWithResponse(ctx, projectName,
		policyID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting OS Update policy %s", policyID))
}
