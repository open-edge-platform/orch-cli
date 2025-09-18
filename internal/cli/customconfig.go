// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
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

var CustomConfigHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Resource ID", "Description")

// Prints OS Profiles in tabular format
func printCustomConfigs(writer io.Writer, CustomConfig []infra.CustomConfigResource, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\t%s\n", "Name", "Resource ID", "Description", "Creation Timestamp", "Updated Timestamp")
	}
	for _, cinit := range CustomConfig {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", cinit.Name, *cinit.ResourceId, *cinit.Description)
		} else {

			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", cinit.Name, *cinit.ResourceId, *cinit.Description, cinit.Timestamps.CreatedAt, cinit.Timestamps.UpdatedAt)
		}
	}
}

// Prints output details of OS Profiles
func printCustomConfig(writer io.Writer, CustomConfig *infra.CustomConfigResource) {

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", CustomConfig.Name)
	_, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *CustomConfig.ResourceId)
	_, _ = fmt.Fprintf(writer, "Description: \t%s\n\n", *CustomConfig.Description)
	_, _ = fmt.Fprintf(writer, "Cloud Init:\n%s\n", CustomConfig.Config)
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
		Aliases: []string{"customconfig", "customconfigs"},
		RunE:    runGetCustomConfigCommand,
	}
	return cmd
}

func getListCustomConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "customconfig [flags]",
		Short:   "List all Cloud Init configurations",
		Example: listCustomConfigExamples,
		Aliases: []string{"customconfig", "customconfigs"},
		RunE:    runListCustomConfigCommand,
	}
	cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of cloud init list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter <filter> see https://google.aip.dev/160 and API spec.")
	return cmd
}

func getCreateCustomConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "customconfig  [flags]",
		Short:   "Creates Cloud Init configuration",
		Example: createCustomConfigExamples,
		Args:    cobra.ExactArgs(2),
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
		RunE:    runDeleteCustomConfigCommand,
	}
	return cmd
}

// Gets specific Cloud Init configuration bu resource ID
func runGetCustomConfigCommand(cmd *cobra.Command, args []string) error {

	writer, verbose := getOutputContext(cmd)
	ctx, customConfigClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	//Leaving this as an example to get by resource ID instead of name
	//CIID := args[0]
	// resp, err := customConfigClient.CustomConfigServiceGetCustomConfigWithResponse(ctx, projectName,
	// 	CIID, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	resp, err := customConfigClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
		&infra.CustomConfigServiceListCustomConfigsParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting Cloud Init config"); !proceed {
		return err
	}

	name := args[0]
	cConfig, err := filterCustomConfigsByName(resp.JSON200.CustomConfigs, name)
	if err != nil {
		return err
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting Cloud Init configuration"); !proceed {
		return err
	}

	printCustomConfig(writer, cConfig)
	return writer.Flush()
}

// Lists all Cloud Init configurations - retrieves all configurations and displays selected information in tabular format
func runListCustomConfigCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	filtflag, _ := cmd.Flags().GetString("filter")
	filter := filterHelper(filtflag)

	ctx, customConfigClient, projectName, err := InfraFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := customConfigClient.CustomConfigServiceListCustomConfigsWithResponse(ctx, projectName,
		&infra.CustomConfigServiceListCustomConfigsParams{
			Filter: filter,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		CustomConfigHeader, "error getting Cloud Init configurations"); !proceed {
		return err
	}

	printCustomConfigs(writer, resp.JSON200.CustomConfigs, verbose)

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
	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error while creating Cloud Init config from %s", path))
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

	if err = checkResponse(gresp.HTTPResponse, "Error getting custom configs"); err != nil {
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

	return checkResponse(resp.HTTPResponse, fmt.Sprintf("error deleting Cloud Init config %s", name))
}
