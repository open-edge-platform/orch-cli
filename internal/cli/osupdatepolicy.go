// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xeipuuv/gojsonschema"
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

var osUpdatePolicySchema = `
{
  "type": "object",
  "properties": {
    "spec": {
      "type": "object",
      "properties": {
        "name":                  { "type": "string" },
        "description":           { "type": "string" },
        "updatePackages":        { "type": "string" },
        "updateKernelCommand":   { "type": "string" },
        "targetOs":              { "type": "string" },
        "updateSources":         { "type": ["array", "null"], "items": { "type": "string" } },
        "updatePolicy":          { "type": "string" }
      },
      "required": ["name", "description", "updatePolicy"]
    }
  },
  "required": ["spec"]
}
`

type OSUpdatePolicy struct {
	Name                string   `yaml:"name"`
	Description         string   `yaml:"description"`
	UpdatePackages      string   `yaml:"updatePackages"`
	UpdateKernelCommand string   `yaml:"updateKernelCommand"`
	TargetOS            string   `yaml:"targetOs"`
	UpdateSources       []string `yaml:"updateSources"`
	UpdatePolicy        string   `yaml:"updatePolicy"`
}

type UpdateNestedSpec struct {
	Spec OSUpdatePolicy `yaml:"spec"`
}

// Template-based output constants for standardization
const (
	DEFAULT_OSUPDATEPOLICY_FORMAT = "table{{.Name}}\t{{str .ResourceId}}\t{{str .Description}}"
	// Use raw timestamp fields in the verbose table so header extraction
	// can detect the field names (fmttime/deref hides them from the extractor).
	DEFAULT_OSUPDATEPOLICY_VERBOSE_FORMAT = "table{{.Name}}\t{{str .ResourceId}}\t{{str .TargetOsId}}\t{{str .Description}}\t{{.Timestamps.CreatedAt}}\t{{.Timestamps.UpdatedAt}}"
	DEFAULT_OSUPDATEPOLICY_GET_FORMAT     = "Name:\t{{.Name}}\nResource ID:\t{{str .ResourceId}}\nTarget OS ID:\t{{str .TargetOsId}}\nTarget OS Name:\t{{if .TargetOs}}{{.TargetOs.Name}}{{end}}\nKernel Command:\t{{str .UpdateKernelCommand}}\nDescription:\t{{str .Description}}\nUpdate Packages:\t{{str .UpdatePackages}}\nUpdate Policy:\t{{deref .UpdatePolicy}}\nUpdate Sources:\t{{deref .UpdateSources}}\nCreated at:\t{{fmttime (deref .Timestamps.CreatedAt)}}\nUpdated at:\t{{fmttime (deref .Timestamps.UpdatedAt)}}\n"
	OSUPDATEPOLICY_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_OSUPDATEPOLICY_OUTPUT_TEMPLATE"
)

// Template-based print helpers for standardization
func getOSUpdatePolicyOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	if verbose && forList {
		return DEFAULT_OSUPDATEPOLICY_VERBOSE_FORMAT, nil
	}
	if !forList {
		// For single-get, return the detailed get format but allow overrides via flags/env
		return resolveTableOutputTemplate(cmd, DEFAULT_OSUPDATEPOLICY_GET_FORMAT, OSUPDATEPOLICY_OUTPUT_TEMPLATE_ENVVAR)
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_OSUPDATEPOLICY_FORMAT, OSUPDATEPOLICY_OUTPUT_TEMPLATE_ENVVAR)
}

func printOSUpdatePolicies(cmd *cobra.Command, writer io.Writer, policies []infra.OSUpdatePolicy, orderBy *string, outputFilter *string, verbose bool) error {
	outputFormat, err := getOSUpdatePolicyOutputFormat(cmd, verbose, true)
	if err != nil {
		return err
	}
	outputType, _ := cmd.Flags().GetString("output-type")
	sortSpec := ""
	if outputType == "table" && orderBy != nil {
		sortSpec = *orderBy
	}
	filterSpec := ""
	if outputType == "table" && outputFilter != nil && *outputFilter != "" {
		filterSpec = *outputFilter
	}
	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OrderBy:   sortSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      policies,
	}
	GenerateOutput(writer, &result)
	return nil
}

func printOSUpdatePolicy(cmd *cobra.Command, writer io.Writer, policy *infra.OSUpdatePolicy) error {
	outputType, _ := cmd.Flags().GetString("output-type")
	outputFormat, err := getOSUpdatePolicyOutputFormat(cmd, false, false)
	if err != nil {
		return err
	}
	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      policy,
	}
	GenerateOutput(writer, &result)
	return nil
}

// Helper function to verify that the input file exists and is of right format
func verifyUpdateProfileInput(path string) error {

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" {
		return errors.New("update profile input must be a yaml file")
	}

	return nil
}

// Helper function to unmarshal yaml file
func readUpdateProfileFromYaml(path string) (*UpdateNestedSpec, error) {
	var input UpdateNestedSpec
	if err := isSafePath(path); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 { // 1MB limit
		return nil, fmt.Errorf("YAML file too large")
	}

	// Unmarshal YAML to map[interface{}]interface{}
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML: %v", err)
	}

	// Convert to map[string]interface{}
	converted := toStringKeyMap(raw)

	// Marshal to JSON for schema validation
	jsonData, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("error converting YAML to JSON: %v", err)
	}
	documentLoader := gojsonschema.NewBytesLoader(jsonData)
	schemaLoader := gojsonschema.NewStringLoader(osUpdatePolicySchema)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %v", err)
	}
	if !result.Valid() {
		var sb strings.Builder
		for _, desc := range result.Errors() {
			sb.WriteString(fmt.Sprintf("- %s\n", desc))
		}
		return nil, fmt.Errorf("YAML does not conform to schema:\n%s", sb.String())
	}

	// Unmarshal YAML to struct after validation
	if err := yaml.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML to struct: %v", err)
	}

	return &input, nil
}

func getGetOSUpdatePolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdatepolicy <name> [flags]",
		Short:   "Get an OS Update policy",
		Example: getOSUpdatePolicyExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: osUpdatePolicyAliases,
		RunE:    runGetOSUpdatePolicyCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getListOSUpdatePolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdatepolicy [flags]",
		Short:   "List all OS Update policies",
		Example: listOSUpdatePolicyExamples,
		Aliases: osUpdatePolicyAliases,
		RunE:    runListOSUpdatePolicyCommand,
	}
	cmd.Flags().StringP("filter", "f", viper.GetString("filter"), "API filter (see https://google.aip.dev/160)")
	cmd.Flags().String("order-by", "", "order results by field (table output only)")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getCreateOSUpdatePolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osupdatepolicy  [flags]",
		Short:   "Creates OS Update policy",
		Example: createOSUpdatePolicyExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: osUpdatePolicyAliases,
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
		Aliases: osUpdatePolicyAliases,
		RunE:    runDeleteOSUpdatePolicyCommand,
	}
	return cmd
}

// Gets specific OSUpdatePolicy - retrieves list of policies and then filters and outputs
// specifc policy by name
func runGetOSUpdatePolicyCommand(cmd *cobra.Command, args []string) error {

	writer, _ := getOutputContext(cmd)
	ctx, OSUpdatePolicyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}
	policyID := args[0]
	resp, err := OSUpdatePolicyClient.OSUpdatePolicyGetOSUpdatePolicyWithResponse(ctx, projectName, policyID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, "error getting OS Update Policy"); err != nil {
		return err
	}
	if err := printOSUpdatePolicy(cmd, writer, resp.JSON200); err != nil {
		return err
	}
	return writer.Flush()
}

// Lists all OS Update policies - retrieves all policies and displays selected information in tabular format
func runListOSUpdatePolicyCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	// filter helper not needed; validation uses API probe
	ctx, OSUPolicyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}
	// Determine validated order-by for API-side ordering when output is json/yaml
	outputType, _ := cmd.Flags().GetString("output-type")
	var validatedOrderBy *string
	if outputType != "table" {
		validatedOrderBy, err = getValidatedOSUpdatePolicyOrderBy(ctx, cmd, OSUPolicyClient, projectName)
		if err != nil {
			return err
		}
	}

	validatedFilter, err := getValidatedOSUpdatePolicyFilter(ctx, cmd, OSUPolicyClient, projectName)
	if err != nil {
		return err
	}

	resp, err := OSUPolicyClient.OSUpdatePolicyListOSUpdatePolicyWithResponse(ctx, projectName,
		&infra.OSUpdatePolicyListOSUpdatePolicyParams{
			Filter:  validatedFilter,
			OrderBy: validatedOrderBy,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting OS Update Policies"); !proceed {
		return err
	}
	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printOSUpdatePolicies(cmd, writer, resp.JSON200.OsUpdatePolicies, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}
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

	ctx, OSUPolicyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	var profile *infra.OperatingSystemResource
	var profileID *string
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
			"", "error getting OS Profile"); !proceed {
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

	if spec.Spec.UpdatePackages != "" {
		packages = &spec.Spec.UpdatePackages
	}

	if spec.Spec.UpdateKernelCommand != "" {
		kernel = &spec.Spec.UpdateKernelCommand
	}

	if spec.Spec.UpdateSources != nil {
		sources = &spec.Spec.UpdateSources
	}

	if profile != nil && profile.ResourceId != nil {
		profileID = profile.ResourceId
	}

	//Create policy
	resp, err := OSUPolicyClient.OSUpdatePolicyCreateOSUpdatePolicyWithResponse(ctx, projectName,
		infra.OSUpdatePolicyCreateOSUpdatePolicyJSONRequestBody{
			Name:                spec.Spec.Name,
			Description:         &spec.Spec.Description,
			UpdatePackages:      packages,
			UpdateKernelCommand: kernel,
			TargetOsId:          profileID,
			UpdateSources:       sources,
			UpdatePolicy:        &updpol,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating OS Update Profiles from %s", path))
}

// Deletes OS Update Policy - checks if a policy  already exists and then deletes it if it does
func runDeleteOSUpdatePolicyCommand(cmd *cobra.Command, args []string) error {

	ctx, OSUPolicyClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	policyID := args[0]

	resp, err := OSUPolicyClient.OSUpdatePolicyDeleteOSUpdatePolicyWithResponse(ctx, projectName,
		policyID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting OS Update policy %s", policyID))
}

// Validates the order-by argument for OSUpdatePolicy and provides hints for valid fields
func getValidatedOSUpdatePolicyOrderBy(ctx interface{}, cmd *cobra.Command, OSUPolicyClient infra.ClientWithResponsesInterface, projectName string) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}
	outputType, _ := cmd.Flags().GetString("output-type")
	// For table format (default), use client-side sorting which supports any field in the model
	if outputType == "table" {
		normalized, _ := normalizeOrderByForClientSorting(raw, infra.OSUpdatePolicy{})
		return normalized, nil
	}
	// For JSON/YAML, use API ordering (only API-supported fields)
	return normalizeOrderByWithAPIProbe(raw, "os-update-policies", infra.OSUpdatePolicy{}, func(orderBy string) (bool, error) {
		pageSize := 1
		offset := 0
		resp, err := OSUPolicyClient.OSUpdatePolicyListOSUpdatePolicyWithResponse(ctx.(context.Context), projectName,
			&infra.OSUpdatePolicyListOSUpdatePolicyParams{
				OrderBy:  &orderBy,
				Filter:   nil,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == 400 {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating OS Update Policy order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func getValidatedOSUpdatePolicyFilter(
	ctx context.Context,
	cmd *cobra.Command,
	OSUPolicyClient infra.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return normalizeFilterWithAPIProbe(raw, "os-update-policies", infra.OSUpdatePolicy{}, func(filter string) (bool, error) {
		pageSize := 1
		offset := 0
		resp, err := OSUPolicyClient.OSUpdatePolicyListOSUpdatePolicyWithResponse(ctx,
			projectName,
			&infra.OSUpdatePolicyListOSUpdatePolicyParams{
				OrderBy:  nil,
				Filter:   &filter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == 400 {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating OS Update Policy filter"); err != nil {
			return false, err
		}
		return true, nil
	})
}
