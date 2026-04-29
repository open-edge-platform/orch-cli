// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	DEFAULT_CUSTOMCONFIG_FORMAT              = "table{{.Name}}\t{{str .ResourceId}}\t{{str .Description}}"
	DEFAULT_CUSTOMCONFIG_LIST_VERBOSE_FORMAT = "table{{.Name}}\t{{str .ResourceId}}\t{{str .Description}}\t{{.Timestamps.CreatedAt}}\t{{.Timestamps.UpdatedAt}}"
	DEFAULT_CUSTOMCONFIG_GET_FORMAT          = "Name: \t{{.Name}}\nResource ID: \t{{str .ResourceId}}\nDescription: \t{{str .Description}}\nCloud Init: \t{{.Config}}\n"
	CUSTOMCONFIG_OUTPUT_TEMPLATE_ENVVAR      = "ORCH_CLI_CUSTOMCONFIG_OUTPUT_TEMPLATE"
)

const listCustomConfigExamples = `# List all custom config (Cloud Init) resources
orch-cli list customconfig --project some-project
`

const getCustomConfigExamples = `# Get detailed information about specific custom config (Cloud Init) resource using it's name
orch-cli get customconfig myconfig --project some-project`

const createCustomConfigExamples = `# Create a custom config (Cloud Init) resource with a given name using cloud init file as input
orch-cli create customconfig myconfig /path/to/cloudinit.yaml  --project some-project

# Create a Cloud Init resource with an optional description 
orch-cli create customconfig myconfig /path/to/cloudinit.yaml  --project some-project --description "This is a cloud init"`

const deleteCustomConfigExamples = `#Delete a custom config (Cloud Init) resource using it's name
orch-cli delete customconfig myconfig --project some-project`

func printCustomConfigs(cmd *cobra.Command, writer io.Writer, customConfigs *[]infra.CustomConfigResource, orderBy *string, outputFilter *string, verbose bool, forList bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getCustomConfigOutputFormat(cmd, verbose, forList)
	if err != nil {
		return err
	}

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
		Data:      *customConfigs,
	}

	GenerateOutput(writer, &result)
	return nil
}

func getCustomConfigOutputFormat(cmd *cobra.Command, verbose bool, forList bool) (string, error) {
	if verbose && forList {
		return DEFAULT_CUSTOMCONFIG_LIST_VERBOSE_FORMAT, nil
	}
	if !forList {
		// Get command always shows full details
		return DEFAULT_CUSTOMCONFIG_GET_FORMAT, nil
	}

	return resolveTableOutputTemplate(cmd, DEFAULT_CUSTOMCONFIG_FORMAT, CUSTOMCONFIG_OUTPUT_TEMPLATE_ENVVAR)
}

// Helper function to verify that the input file exists and is of right format
func verifyName(n string) error {

	pattern := `^[a-zA-Z0-9_\-]`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Match the input string against the pattern
	if re.MatchString(n) {
		return nil
	}
	return errors.New("input is not an alphanumeric single word")
}

// readCustomConfigFromYaml reads the contents of a YAML file and returns it as a string.
func readCustomConfigFromYaml(path string) (string, error) {

	if err := isSafePath(path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) > 1<<20 { // 1MB limit
		return "", fmt.Errorf("YAML file too large")
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "#cloud-config") {
		return "", fmt.Errorf("file does not start with #cloud-config")
	}
	var out interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("invalid YAML: %w", err)
	}
	return string(data), nil
}

// Filters list of pcustom configs to find one with specific name
func filterCustomConfigsByName(CustomConfigs []infra.CustomConfigResource, name string) (*infra.CustomConfigResource, error) {
	for _, config := range CustomConfigs {
		if config.Name == name {
			return &config, nil
		}
	}
	return nil, errors.New("no custom config matches the given name")
}

func getGetCustomConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "customconfig <name> [flags]",
		Short:   "Get a Cloud Init configuration",
		Example: getCustomConfigExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: customConfigAliases,
		RunE:    runGetCustomConfigCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getListCustomConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "customconfig [flags]",
		Short:   "List all Cloud Init configurations",
		Example: listCustomConfigExamples,
		Aliases: customConfigAliases,
		RunE:    runListCustomConfigCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "customconfig")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getCreateCustomConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "customconfig  [flags]",
		Short:   "Creates Cloud Init configuration",
		Example: createCustomConfigExamples,
		Args:    cobra.ExactArgs(2),
		Aliases: customConfigAliases,
		RunE:    runCreateCustomConfigCommand,
	}
	cmd.PersistentFlags().StringP("description", "d", viper.GetString("description"), "Optional flag used to provide a description to a cloud init config resource")
	return cmd
}

func getDeleteCustomConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "customconfig <name> [flags]",
		Short:   "Delete a Cloud Init config",
		Example: deleteCustomConfigExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: customConfigAliases,
		RunE:    runDeleteCustomConfigCommand,
	}
	return cmd
}

// Gets specific Cloud Init configuration bu resource ID
func runGetCustomConfigCommand(cmd *cobra.Command, args []string) error {
	writer, _ := getOutputContext(cmd)
	ctx, customConfigClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := customConfigClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
		&infra.CustomConfigServiceListCustomConfigsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting Cloud Init config"); !proceed {
		return err
	}

	name := args[0]
	cConfig, err := filterCustomConfigsByName(resp.JSON200.CustomConfigs, name)
	if err != nil {
		return err
	}

	customConfigs := []infra.CustomConfigResource{*cConfig}
	var emptyFilter string
	// Get command always shows full details (forList=false)
	if err := printCustomConfigs(cmd, writer, &customConfigs, nil, &emptyFilter, false, false); err != nil {
		return err
	}

	return writer.Flush()
}

// Lists all Cloud Init configurations - retrieves all configurations and displays selected information in tabular format
func runListCustomConfigCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, customConfigClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")

	var validatedOrderBy *string
	if outputType == "table" {
		validatedOrderBy, err = normalizeOrderByForClientSorting(raw, infra.CustomConfigResource{})
	} else {
		validatedOrderBy, err = normalizeOrderByWithAPIProbe(raw, "customconfig", infra.CustomConfigResource{}, func(orderBy string) (bool, error) {
			pageSize := 1
			resp, err := customConfigClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
				&infra.CustomConfigServiceListCustomConfigsParams{
					OrderBy:  &orderBy,
					PageSize: &pageSize,
				}, auth.AddAuthHeader)
			if err != nil {
				return false, processError(err)
			}
			if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusBadRequest {
				return false, &api400Error{string(resp.Body)}
			}
			if err := checkResponse(resp.HTTPResponse, resp.Body, "error validating customconfig order-by"); err != nil {
				return false, err
			}
			return true, nil
		})
	}
	if err != nil {
		return err
	}

	apiOrderBy := validatedOrderBy
	if outputType == "table" {
		// Table output sorts locally via GenerateOutput(CommandResult.OrderBy).
		apiOrderBy = nil
	}

	pageSize32, offset32, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	pageSize := int(pageSize32)
	offset := int(offset32)
	if pageSize <= 0 {
		pageSize = 100
	}

	// Preserve explicit pagination requests as single-page results.
	if cmd.Flags().Changed("page-size") || cmd.Flags().Changed("offset") {
		params := &infra.CustomConfigServiceListCustomConfigsParams{
			OrderBy:  apiOrderBy,
			Filter:   getNonEmptyFlag(cmd, "filter"),
			PageSize: &pageSize,
			Offset:   &offset,
		}

		resp, err := customConfigClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName, params, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}

		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
			"", "error getting Cloud Init configurations"); !proceed {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.CustomConfigs == nil {
			return fmt.Errorf("error listing custom configs: unexpected response format")
		}

		customConfigs := resp.JSON200.CustomConfigs

		outputFilter, _ := cmd.Flags().GetString("output-filter")
		if err := printCustomConfigs(cmd, writer, &customConfigs, validatedOrderBy, &outputFilter, verbose, true); err != nil {
			return err
		}
		return writer.Flush()
	}

	// Automatic pagination: fetch all pages
	allCustomConfigs := make([]infra.CustomConfigResource, 0)

	resp, err := customConfigClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
		&infra.CustomConfigServiceListCustomConfigsParams{
			OrderBy:  apiOrderBy,
			Filter:   getNonEmptyFlag(cmd, "filter"),
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting Cloud Init configurations"); !proceed {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.CustomConfigs == nil {
		return fmt.Errorf("error listing custom configs: unexpected response format")
	}

	allCustomConfigs = append(allCustomConfigs, resp.JSON200.CustomConfigs...)
	totalElements := int(resp.JSON200.TotalElements)

	// When page size is omitted (0), derive increment from the first page length.
	if pageSize <= 0 {
		pageSize = len(resp.JSON200.CustomConfigs)
	}

	for len(allCustomConfigs) < totalElements {
		if pageSize <= 0 {
			break
		}
		offset += pageSize
		resp, err := customConfigClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
			&infra.CustomConfigServiceListCustomConfigsParams{
				OrderBy:  apiOrderBy,
				Filter:   getNonEmptyFlag(cmd, "filter"),
				PageSize: &pageSize,
				Offset:   &offset,
			}, auth.AddAuthHeader)
		if err != nil {
			return processError(err)
		}
		if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
			"", "error getting Cloud Init configurations"); !proceed {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.CustomConfigs == nil {
			return fmt.Errorf("error listing custom configs: unexpected response format")
		}

		if len(resp.JSON200.CustomConfigs) == 0 {
			break
		}
		allCustomConfigs = append(allCustomConfigs, resp.JSON200.CustomConfigs...)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printCustomConfigs(cmd, writer, &allCustomConfigs, validatedOrderBy, &outputFilter, verbose, true); err != nil {
		return err
	}

	return writer.Flush()
}

// Creates Cloud Init config
func runCreateCustomConfigCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	path := args[1]

	var desc *string
	descFlag, _ := cmd.Flags().GetString("description")
	if descFlag != "" {
		desc = &descFlag
	}

	err := verifyName(name)
	if err != nil {
		return err
	}

	config, err := readCustomConfigFromYaml(path)
	if err != nil {
		return err
	}

	ctx, customConfigClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := customConfigClient.CustomConfigServiceCreateCustomConfigWithResponse(ctx, projectName,
		infra.CustomConfigServiceCreateCustomConfigJSONRequestBody{
			Name:        name,
			Description: desc,
			Config:      config,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating Cloud Init config from %s", path))
}

// Deletes Cloud Init config - checks if a config already exists and then deletes it if it does
func runDeleteCustomConfigCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	ctx, customConfigClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	gresp, err := customConfigClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
		&infra.CustomConfigServiceListCustomConfigsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if err = checkResponse(gresp.HTTPResponse, gresp.Body, "Error getting custom configs"); err != nil {
		return err
	}

	cConfig, err := filterCustomConfigsByName(gresp.JSON200.CustomConfigs, name)
	if err != nil {
		return err
	}

	resp, err := customConfigClient.CustomConfigServiceDeleteCustomConfigWithResponse(ctx, projectName,
		*cConfig.ResourceId, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting Cloud Init config %s", name))
}
