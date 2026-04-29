// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

const (
	DEFAULT_OSPROFILE_FORMAT         = "table{{str .Name}}\t{{str .Architecture}}\t{{.SecurityFeature}}"
	DEFAULT_OSPROFILE_VERBOSE_FORMAT = "Name: \t{{str .Name}}\nProfile Name: \t{{str .ProfileName}}\nSecurity Feature: \t{{.SecurityFeature}}\nArchitecture: \t{{str .Architecture}}\nRepository URL: \t{{str .RepoUrl}}\nsha256: \t{{.Sha256}}\n"
	DEFAULT_OSPROFILE_INSPECT_FORMAT = "Name: \t{{str .Name}}\nProfile Name: \t{{str .ProfileName}}\nOS Resource ID: \t{{str .OsResourceID}}\nVersion: \t{{str .ProfileVersion}}\nSha256: \t{{.Sha256}}\nImage ID: \t{{str .ImageId}}\nImage URL: \t{{str .ImageUrl}}\nRepository URL: \t{{str .RepoUrl}}\nDescription: \t{{str .Description}}\nMetadata: \t{{str .Metadata}}\nSecurity Feature: \t{{.SecurityFeature}}\nArchitecture: \t{{str .Architecture}}\nOS Type: \t{{.OsType}}\nOS Provider: \t{{.OsProvider}}\nPlatform Bundle: \t{{str .PlatformBundle}}\nInstalled Packages: \t{{str .InstalledPackages}}\nCreated: \t{{.Timestamps.CreatedAt}}\nUpdated: \t{{.Timestamps.UpdatedAt}}\n{{if .TlsCaCert}}TLS CA Cert: \t{{str .TlsCaCert}}\n{{end}}{{if .ExistingCves}}Existing CVEs: \t{{str .ExistingCves}}\n{{end}}{{if .FixedCves}}Fixed CVEs: \t{{str .FixedCves}}\n{{end}}"
	OSPROFILE_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_OSPROFILE_OUTPUT_TEMPLATE"
)

const listOSProfileExamples = `# List all OS Profiles
orch-cli list osprofile --project some-project

# List OS Profiles using a custom filter (see: https://google.aip.dev/160 and API spec @ https://github.com/open-edge-platform/orch-utils/blob/main/tenancy-api-mapping/openapispecs/generated/amc-infra-core-edge-infrastructure-manager-openapi-all.yaml )
orch-cli list osprofile --project some-project --filter "osType=OS_TYPE_IMMUTABLE"`

const getOSProfileExamples = `# Get detailed information about specific OS Profile using the os profile name
orch-cli get osprofile osprofilename --project some-project`

const createOSProfileExamples = `# Create an OS Profile using a valid .yaml manifest as an input.
orch-cli create osprofile ./microvisor-nonrt.yaml  --project some-project

Example .yaml manifest:

spec:
  name: Edge Microvisor Toolkit <ver>
  type: OPERATING_SYSTEM_TYPE_IMMUTABLE
  provider: OPERATING_SYSTEM_PROVIDER_INFRA
  architecture: x86_64
  profileName: <profile name>  # Name has to be identical to this file's name
  osImageUrl: files-edge-orch/repository/microvisor/non_rt/<artfact.raw.gz>
  osImageVersion: <version>
  osImageSha256: <sha>
  osPackageManifestURL: files-edge-orch/repository/microvisor/non_rt/<manifest.json>
  securityFeature: SECURITY_FEATURE_NONE
  platformBundle:
  metadata:
	key1: value1
	key2: value2

See 
https://github.com/open-edge-platform/infra-core/tree/main/os-profiles`

const deleteOSProfileExamples = `#Delete an OS Profile using it's name
orch-cli delete osprofile "Edge Microvisor Toolkit 3.0.20250504" --project some-project`

var osProfileSchema = `
{
  "type": "object",
  "properties": {
    "spec": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "type": { "type": "string" },
        "provider": { "type": "string" },
        "architecture": { "type": "string" },
        "profileName": { "type": "string" },
        "osImageUrl": { "type": "string" },
        "osImageSha256": { "type": "string" },
        "osImageVersion": { "type": "string" },
        "osPackageManifestURL": { "type": "string" },
		"existingCvesURL": { "type": ["string", "null"] },
		"fixedCvesURL": { "type": ["string", "null"] },
        "securityFeature": { "type": "string" },
        "platformBundle": { "type": ["string", "null"] },
		"tlsCaCert": { "type": ["string", "null"] },
		"description": { "type": ["string", "null"] },
		"metadata": { 
          "type": ["object", "null"],
          "additionalProperties": true
        }
      },
      "required": [
        "name", "type", "provider", "architecture", "profileName",
        "osImageUrl", "osImageSha256", "osImageVersion",
        "osPackageManifestURL", "securityFeature", "platformBundle"
      ]
    }
  },
  "required": ["spec"]
}
`

type OSProfileSpec struct {
	Name              string                 `yaml:"name"`
	Type              string                 `yaml:"type"`
	Provider          string                 `yaml:"provider"`
	Architecture      string                 `yaml:"architecture"`
	ProfileName       string                 `yaml:"profileName"`
	OsImageURL        string                 `yaml:"osImageUrl"`
	OsImageSha256     string                 `yaml:"osImageSha256"`
	OsImageVersion    string                 `yaml:"osImageVersion"`
	OSPackageURL      string                 `yaml:"osPackageManifestURL"`
	SecurityFeature   string                 `yaml:"securityFeature"`
	PlatformBundle    string                 `yaml:"platformBundle"`
	OsExistingCvesURL string                 `yaml:"osExistingCvesURL"`
	OsFixedCvesURL    string                 `yaml:"osFixedCvesURL"`
	TLSCaCert         string                 `yaml:"tlsCaCert"`
	Description       string                 `yaml:"description"`
	Metadata          map[string]interface{} `yaml:"metadata"`
}

type NestedSpec struct {
	AppVersion string        `yaml:"appVersion"`
	Spec       OSProfileSpec `yaml:"spec"`
}

func getOSProfileOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	if verbose && forList {
		return DEFAULT_OSPROFILE_VERBOSE_FORMAT, nil
	}
	if !forList {
		// Get command always shows full details
		return DEFAULT_OSPROFILE_INSPECT_FORMAT, nil
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_OSPROFILE_FORMAT, OSPROFILE_OUTPUT_TEMPLATE_ENVVAR)
}

// Prints OS Profiles in tabular format
func printOSProfiles(cmd *cobra.Command, writer io.Writer, OSProfiles []infra.OperatingSystemResource, orderBy *string, outputFilter *string, verbose bool) error {
	outputFormat, err := getOSProfileOutputFormat(cmd, verbose, true)
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
		Data:      OSProfiles,
	}

	GenerateOutput(writer, &result)
	return nil
}

// Prints output details of OS Profiles
func printOSProfile(cmd *cobra.Command, writer io.Writer, OSProfile *infra.OperatingSystemResource) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getOSProfileOutputFormat(cmd, false, false)
	if err != nil {
		return err
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      OSProfile,
	}
	GenerateOutput(writer, &result)
	return nil
}

// Helper function to verify that the input file exists and is of right format
func verifyOSProfileInput(path string) error {

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" {
		return errors.New("os Profile input must be a yaml file")
	}

	return nil
}

// Helper function to unmarshal yaml file
func readOSProfileFromYaml(path string) (*NestedSpec, error) {

	var input NestedSpec

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
	schemaLoader := gojsonschema.NewStringLoader(osProfileSchema)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %v", err)
	}
	if !result.Valid() {
		var sb strings.Builder
		for _, desc := range result.Errors() {
			fmt.Fprintf(&sb, "- %s\n", desc)
		}
		return nil, fmt.Errorf("YAML does not conform to schema:\n%s", sb.String())
	}

	// Unmarshal YAML to struct after validation
	if err := yaml.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML to struct: %v", err)
	}

	return &input, nil
}

// Filters list of profiles to find one with specific name
func filterProfilesByName(OSProfiles []infra.OperatingSystemResource, name string) (*infra.OperatingSystemResource, error) {
	for _, profile := range OSProfiles {
		if *profile.Name == name {
			return &profile, nil
		}
	}
	return nil, errors.New("no os profile matches the given name")
}

func getGetOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osprofile <name> [flags]",
		Short:   "Get an OS profile",
		Example: getOSProfileExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: osProfileAliases,
		RunE:    runGetOSProfileCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getListOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osprofile [flags]",
		Short:   "List all OS profiles",
		Example: listOSProfileExamples,
		Aliases: osProfileAliases,
		RunE:    runListOSProfileCommand,
	}
	cmd.Flags().StringP("filter", "f", "", "API filter (see https://google.aip.dev/160)")
	cmd.Flags().String("order-by", "", "order results by field (table output only)")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getCreateOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osprofile </path/to/profile.yaml> [flags]",
		Short:   "Creates OS profile",
		Example: createOSProfileExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: osProfileAliases,
		RunE:    runCreateOSProfileCommand,
	}
	return cmd
}

func getDeleteOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osprofile <name> [flags]",
		Short:   "Delete an OS profile",
		Example: deleteOSProfileExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: osProfileAliases,
		RunE:    runDeleteOSProfileCommand,
	}
	return cmd
}

// Gets specific OS Profile - retrieves list of profiles and then filters and outputs
// specifc profile by name
func runGetOSProfileCommand(cmd *cobra.Command, args []string) error {
	writer, _ := getOutputContext(cmd)
	ctx, OSProfileClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
		&infra.OperatingSystemServiceListOperatingSystemsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err := checkResponse(resp.HTTPResponse, resp.Body, "error getting OS Profile"); err != nil {
		return err
	}

	name := args[0]
	profile, err := filterProfilesByName(resp.JSON200.OperatingSystemResources, name)
	if err != nil {
		return err
	}

	if err := printOSProfile(cmd, writer, profile); err != nil {
		return err
	}
	return writer.Flush()
}

func getValidatedOSProfileOrderBy(
	ctx context.Context,
	cmd *cobra.Command,
	OSProfileClient infra.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return nil, err
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	// For table format (default), use client-side sorting which supports any field in the model
	if outputType == "table" {
		return normalizeOrderByForClientSorting(raw, infra.OperatingSystemResource{})
	}

	// For JSON/YAML, use API ordering (only API-supported fields)
	return normalizeOrderByWithAPIProbe(raw, "os profiles", infra.OperatingSystemResource{}, func(orderBy string) (bool, error) {
		pageSize := int(1)
		offset := int(0)
		// Validate ordering in isolation
		resp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
			&infra.OperatingSystemServiceListOperatingSystemsParams{
				OrderBy:  &orderBy,
				Filter:   nil,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating OS profile order-by"); err != nil {
			return false, err
		}
		return true, nil
	})
}

func getValidatedOSProfileFilter(
	ctx context.Context,
	cmd *cobra.Command,
	OSProfileClient infra.ClientWithResponsesInterface,
	projectName string,
) (*string, error) {
	raw, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return normalizeFilterWithAPIProbe(raw, "os profiles", infra.OperatingSystemResource{}, func(filter string) (bool, error) {
		pageSize := int(1)
		offset := int(0)
		resp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
			&infra.OperatingSystemServiceListOperatingSystemsParams{
				OrderBy:  nil,
				Filter:   &filter,
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return false, processError(err)
		}
		if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
			return false, &api400Error{string(resp.Body)}
		}
		if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating OS profile filter"); err != nil {
			return false, err
		}
		return true, nil
	})
}

// Lists all OS Profiles - retrieves all profiles and displays selected information in tabular format
func runListOSProfileCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	// filter helper not needed; validation uses API probe

	ctx, OSProfileClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	// Get and validate order-by
	validatedOrderBy, err := getValidatedOSProfileOrderBy(ctx, cmd, OSProfileClient, projectName)
	if err != nil {
		return err
	}

	// Determine if we use API or client-side ordering
	outputType, _ := cmd.Flags().GetString("output-type")
	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		apiOrderBy = nil
	}

	validatedFilter, err := getValidatedOSProfileFilter(ctx, cmd, OSProfileClient, projectName)
	if err != nil {
		return err
	}

	resp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
		&infra.OperatingSystemServiceListOperatingSystemsParams{
			Filter:  validatedFilter,
			OrderBy: apiOrderBy,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err := checkResponse(resp.HTTPResponse, resp.Body, "error getting OS Profiles"); err != nil {
		return err
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printOSProfiles(cmd, writer, resp.JSON200.OperatingSystemResources, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}

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

	ctx, OSProfileClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	//TODO Delete name check once API accepts only unique names
	gresp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
		&infra.OperatingSystemServiceListOperatingSystemsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err = checkResponse(gresp.HTTPResponse, gresp.Body, "Error getting OS profiles"); err != nil {
		return err
	}

	_, err = filterProfilesByName(gresp.JSON200.OperatingSystemResources, spec.Spec.Name)
	if err == nil {
		return fmt.Errorf("OS Profile %s already exists", spec.Spec.Name)
	}
	// End TODO

	metadataJSON, err := convertMetadataToAPIString(spec.Spec.Metadata)
	if err != nil {
		return fmt.Errorf("metadata validation failed: %v", err)
	}

	resp, err := OSProfileClient.OperatingSystemServiceCreateOperatingSystemWithResponse(ctx, projectName,
		infra.OperatingSystemServiceCreateOperatingSystemJSONRequestBody{
			Name:            &spec.Spec.Name,
			Architecture:    &spec.Spec.Architecture,
			ImageUrl:        &spec.Spec.OsImageURL,
			ImageId:         &spec.Spec.OsImageVersion,
			OsType:          (*infra.OsType)(&spec.Spec.Type),
			OsProvider:      (*infra.OsProviderKind)(&spec.Spec.Provider),
			ProfileName:     &spec.Spec.ProfileName,
			RepoUrl:         &spec.Spec.OsImageURL,
			SecurityFeature: (*infra.SecurityFeature)(&spec.Spec.SecurityFeature),
			Sha256:          spec.Spec.OsImageSha256,
			FixedCvesUrl:    &spec.Spec.OsFixedCvesURL,
			ExistingCvesUrl: &spec.Spec.OsExistingCvesURL,
			TlsCaCert:       &spec.Spec.TLSCaCert,
			Description:     &spec.Spec.Description,
			Metadata:        metadataJSON,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating OS Profile from %s", path))
}

// Deletes OS Profile - checks if a profile already exists and then deletes it if it does
func runDeleteOSProfileCommand(cmd *cobra.Command, args []string) error {
	ctx, OSProfileClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	gresp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
		&infra.OperatingSystemServiceListOperatingSystemsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err = checkResponse(gresp.HTTPResponse, gresp.Body, "Error getting OS profiles"); err != nil {
		return err
	}

	name := args[0]
	profile, err := filterProfilesByName(gresp.JSON200.OperatingSystemResources, name)
	if err != nil {
		return err
	}

	resp, err := OSProfileClient.OperatingSystemServiceDeleteOperatingSystemWithResponse(ctx, projectName,
		*profile.OsResourceID, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting OS profile %s", name))
}

// Converts map[interface{}]interface{} to map[string]interface{} recursively
func toStringKeyMap(m interface{}) interface{} {
	switch x := m.(type) {
	case map[interface{}]interface{}:
		n := make(map[string]interface{})
		for k, v := range x {
			n[fmt.Sprintf("%v", k)] = toStringKeyMap(v)
		}
		return n
	case []interface{}:
		for i, v := range x {
			x[i] = toStringKeyMap(v)
		}
	}
	return m
}

func convertMetadataToAPIString(metadata map[string]interface{}) (*string, error) {
	if len(metadata) == 0 {
		return nil, nil
	}

	// Convert metadata map to JSON string
	jsonBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("error converting metadata to JSON: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Check length constraint (500 characters)
	if len(jsonStr) > 500 {
		return nil, fmt.Errorf("metadata JSON exceeds maximum length of 500 characters (got %d): %s", len(jsonStr), jsonStr)
	}

	// Validate against API regex pattern: ^$|^[a-z0-9,.\-_:/"\\ \\n{}\[\]]+$
	// This will fail if the input contains invalid characters - no sanitization
	for i, char := range jsonStr {
		valid := false
		switch {
		case char >= 'a' && char <= 'z':
			valid = true
		case char >= '0' && char <= '9':
			valid = true
		case char == ',' || char == '.' || char == '-' || char == '_' ||
			char == ':' || char == '/' || char == '"' || char == '\\' ||
			char == ' ' || char == '\n' || char == '{' || char == '}' ||
			char == '[' || char == ']':
			valid = true
		}

		if !valid {
			return nil, fmt.Errorf("metadata contains invalid character '%c' at position %d. API only allows: a-z, 0-9, comma, period, dash, underscore, colon, forward slash, quotes, backslash, space, newline, curly braces, square brackets", char, i)
		}
	}

	return &jsonStr, nil
}
