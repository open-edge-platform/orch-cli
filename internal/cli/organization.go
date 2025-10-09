// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/tenancy"
	"github.com/spf13/cobra"
)

const listOrganizationExamples = `# List all ogranizations in the organization
orch-cli list ogranizations
`

const getOrganizationExamples = `# Get detailed information about specific ogranization
orch-cli get ogranization myogranization
`

const createOrganizationExamples = `# Create a ogranization with a given name using cloud init file as input
orch-cli create ogranization myogranization`

const deleteOrganizationExamples = `#Delete a ogranization using it's name
orch-cli delete ogranization myogranization`

var OrganizationHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Resource ID", "Description")

// Prints OS Profiles in tabular format
func printOrganizations(writer io.Writer, organizations *tenancy.OrgOrgList, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\t%s\n", "Name", "Resource ID", "Description", "Creation Timestamp", "Updated Timestamp")
	}
	// for _, ogranization := range ogranizations {
	// 	if !verbose {
	// 		fmt.Fprintf(writer, "%s\t%s\t%s\n", ogranization, *ogranization.ResourceId, *ogranization.Description)
	// 	} else {

	// 		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", ogranization.Name, *ogranization.ResourceId, *ogranization.Description, ogranization.Timestamps.CreatedAt, ogranization.Timestamps.UpdatedAt)
	// 	}
	// }
}

// Prints output details of OS Profiles
func printOrganization(writer io.Writer, ogranization *tenancy.GetorgOrg) {

	// _, _ = fmt.Fprintf(writer, "Name: \t%s\n", ogranization.Name)
	// _, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *ogranization.ResourceId)
	// _, _ = fmt.Fprintf(writer, "Description: \t%s\n\n", *ogranization.Description)
	// _, _ = fmt.Fprintf(writer, "Cloud Init:\n%s\n", ogranization.Config)
}

// // Filters list of pcustom configs to find one with specific name
// func filterCustomConfigsByName(CustomConfigs []infra.CustomConfigResource, name string) (*infra.CustomConfigResource, error) {
// 	for _, config := range CustomConfigs {
// 		if config.Name == name {
// 			return &config, nil
// 		}
// 	}
// 	return nil, errors.New("no custom config matches the given name")
// }

func getGetOrganizationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ogranization <name> [flags]",
		Short:   "Get a ogranization",
		Example: getOrganizationExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: organizationAliases,
		RunE:    runGetOrganizationCommand,
	}
	return cmd
}

func getListOrganizationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "organization [flags]",
		Short:   "List all organizations",
		Example: listOrganizationExamples,
		Aliases: organizationAliases,
		RunE:    runListOrganizationCommand,
	}
	//cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of cloud init list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter <filter> see https://google.aip.dev/160 and API spec.")
	return cmd
}

func getCreateOrganizationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "organization [flags]",
		Short:   "Creates a organization",
		Example: createOrganizationExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: organizationAliases,
		RunE:    runCreateOrganizationCommand,
	}
	//cmd.PersistentFlags().StringP("description", "d", viper.GetString("description"), "Optional flag used to provide a description to a cloud init config resource")
	return cmd
}

func getDeleteOrganizationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "organization <name> [flags]",
		Short:   "Delete a organization",
		Example: deleteOrganizationExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: organizationAliases,
		RunE:    runDeleteOrganizationCommand,
	}
	return cmd
}

// Gets specific Cloud Init configuration bu resource ID
func runGetOrganizationCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	writer, verbose := getOutputContext(cmd)
	ctx, ogranizationClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	//Leaving this as an example to get by resource ID instead of name
	//CIID := args[0]
	// resp, err := customConfigClient.CustomConfigServiceGetCustomConfigWithResponse(ctx, ogranizationName,
	// 	CIID, auth.AddAuthHeader)
	// if err != nil {
	// 	return processError(err)
	// }

	resp, err := ogranizationClient.GETV1OrgsOrgOrgWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting ogranizations"); !proceed {
		return err
	}

	printOrganization(writer, resp.JSON200)
	return writer.Flush()
}

// Lists all Cloud Init configurations - retrieves all configurations and displays selected information in tabular format
func runListOrganizationCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, ogranizationClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := ogranizationClient.LISTV1OrgsWithResponse(ctx, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OrganizationHeader, "error getting ogranizations"); !proceed {
		return err
	}

	printOrganizations(writer, resp.JSON200, verbose)

	return writer.Flush()
}

// Creates Organization
func runCreateOrganizationCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	var desc *string
	descFlag, _ := cmd.Flags().GetString("description")
	if descFlag != "" {
		desc = &descFlag
	}

	ctx, ogranizationClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := ogranizationClient.PUTV1OrgsOrgOrgWithResponse(ctx, name, &tenancy.PUTV1OrgsOrgOrgParams{},
		tenancy.PUTV1OrgsOrgOrgJSONRequestBody{
			Description: desc,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating projet"))
}

// Deletes Organization - checks if a ogranization already exists and then deletes it if it does
func runDeleteOrganizationCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	ctx, ogranizationClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := ogranizationClient.DELETEV1OrgsOrgOrgWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting organization %s", name))
}
