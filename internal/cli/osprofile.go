// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
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
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
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

See 
https://github.com/open-edge-platform/infra-core/tree/main/os-profiles`

const deleteOSProfileExamples = `#Delete an OS Profile using it's name
orch-cli delete osprofile "Edge Microvisor Toolkit 3.0.20250504" --project some-project`

var OSProfileHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Architecture", "Security Feature")
var OSProfileHeaderGet = fmt.Sprintf("\n%s\t%s", "OS Profile Field", "Value")

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
        "platformBundle": { "type": ["string", "null"] }
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
	Name              string `yaml:"name"`
	Type              string `yaml:"type"`
	Provider          string `yaml:"provider"`
	Architecture      string `yaml:"architecture"`
	ProfileName       string `yaml:"profileName"`
	OsImageURL        string `yaml:"osImageUrl"`
	OsImageSha256     string `yaml:"osImageSha256"`
	OsImageVersion    string `yaml:"osImageVersion"`
	OSPackageURL      string `yaml:"osPackageManifestURL"`
	SecurityFeature   string `yaml:"securityFeature"`
	PlatformBundle    string `yaml:"platformBundle"`
	OsExistingCvesURL string `yaml:"osExistingCvesURL"`
	OsFixedCvesURL    string `yaml:"osFixedCvesURL"`
}

type NestedSpec struct {
	AppVersion string        `yaml:"appVersion"`
	Spec       OSProfileSpec `yaml:"spec"`
}

// Prints OS Profiles in tabular format
func printOSProfiles(writer io.Writer, OSProfiles []infra.OperatingSystemResource, verbose bool) {
	for _, osp := range OSProfiles {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *osp.Name, *osp.Architecture, *osp.SecurityFeature)
		} else {
			_, _ = fmt.Fprintf(writer, "\nName:\t %s\n", *osp.Name)
			_, _ = fmt.Fprintf(writer, "Profile Name:\t %s\n", *osp.ProfileName)
			_, _ = fmt.Fprintf(writer, "Security Feature:\t %v\n", toJSON(osp.SecurityFeature))
			_, _ = fmt.Fprintf(writer, "Architecture:\t %s\n", *osp.Architecture)
			_, _ = fmt.Fprintf(writer, "Repository URL:\t %s\n", *osp.RepoUrl)
			_, _ = fmt.Fprintf(writer, "sha256:\t %v\n", osp.Sha256)
			_, _ = fmt.Fprintf(writer, "Kernel Command:\t %v\n", toJSON(osp.KernelCommand))
		}
	}
}

// Prints output details of OS Profiles
func printOSProfile(writer io.Writer, OSProfile *infra.OperatingSystemResource) {
	var cveEntries []CVEEntry
	var fcveEntries []CVEEntry
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

	if OSProfile.ExistingCves != nil && OSProfile.FixedCves != nil {

		if *OSProfile.ExistingCves != "" {
			err := json.Unmarshal([]byte(*OSProfile.ExistingCves), &cveEntries)
			if err != nil {
				fmt.Println("Error unmarshaling JSON: existing CVE entries:", err)
				return
			}
		}
		if *OSProfile.FixedCves != "" {
			err := json.Unmarshal([]byte(*OSProfile.FixedCves), &fcveEntries)
			if err != nil {
				fmt.Println("Error unmarshaling JSON: fixed CVE entries:", err)
				return
			}
		}

		_, _ = fmt.Fprintf(writer, "\nCVE Info:\n")
		_, _ = fmt.Fprintf(writer, "\t Existing CVEs: \n\n")
		for _, cve := range cveEntries {
			_, _ = fmt.Fprintf(writer, "-\t\tCVE ID:\t %v\n", cve.CVEID)
			_, _ = fmt.Fprintf(writer, "-\t\tPriority:\t %v\n", cve.Priority)
			_, _ = fmt.Fprintf(writer, "-\t\tAffected Packages:\t %v\n\n", cve.AffectedPackages)
		}
		_, _ = fmt.Fprintf(writer, "\t Fixed CVEs: \n\n")
		for _, fcve := range fcveEntries {
			_, _ = fmt.Fprintf(writer, "-\t\tCVE ID:\t %v\n", fcve.CVEID)
			_, _ = fmt.Fprintf(writer, "-\t\tPriority:\t %v\n", fcve.Priority)
			_, _ = fmt.Fprintf(writer, "-\t\tAffected Packages:\t %v\n\n", fcve.AffectedPackages)
		}
	}
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
		RunE:    runGetOSProfileCommand,
	}
	return cmd
}

func getListOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osprofile [flags]",
		Short:   "List all OS profiles",
		Example: listOSProfileExamples,
		RunE:    runListOSProfileCommand,
	}
	cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of host list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter \"osType=OS_TYPE_IMMUTABLE\" see https://google.aip.dev/160 and API spec.")
	return cmd
}

func getCreateOSProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "osprofile </path/to/profile.yaml> [flags]",
		Short:   "Creates OS profile",
		Example: createOSProfileExamples,
		Args:    cobra.ExactArgs(1),
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
		RunE:    runDeleteOSProfileCommand,
	}
	return cmd
}

// Gets specific OS Profile - retrieves list of profiles and then filters and outputs
// specifc profile by name
func runGetOSProfileCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, OSProfileClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
		&infra.OperatingSystemServiceListOperatingSystemsParams{}, auth.AddAuthHeader)
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

	filtflag, _ := cmd.Flags().GetString("filter")
	filter := filterHelper(filtflag)

	ctx, OSProfileClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := OSProfileClient.OperatingSystemServiceListOperatingSystemsWithResponse(ctx, projectName,
		&infra.OperatingSystemServiceListOperatingSystemsParams{
			Filter: filter,
		}, auth.AddAuthHeader)
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

	if err = checkResponse(gresp.HTTPResponse, "Error getting OS profiles"); err != nil {
		return err
	}

	_, err = filterProfilesByName(gresp.JSON200.OperatingSystemResources, spec.Spec.Name)
	if err == nil {
		return fmt.Errorf("OS Profile %s already exists", spec.Spec.Name)
	}
	// End TODO

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
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating OS Profile from %s", path))
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

	if err = checkResponse(gresp.HTTPResponse, "Error getting OS profiles"); err != nil {
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

	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting OS profile %s", name))
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
