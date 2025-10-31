// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/tenancy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listOrganizationExamples = `# List all organizations in the organization
orch-cli list organizations
`

const getOrganizationExamples = `# Get detailed information about specific organization
orch-cli get organization myorganization
`

const createOrganizationExamples = `# Create a organization with a given name 
orch-cli create organization myorganization

# Create a organization with a given name and description
orch-cli create organization myorganization --description "my description"
`

const deleteOrganizationExamples = `#Delete a organization using it's name
orch-cli delete organization myorganization`

var OrganizationHeader = fmt.Sprintf("\n%s\t%s", "Name", "Status")

// Prints OS Profiles in tabular format
func printOrganizations(writer io.Writer, organizations *tenancy.OrgOrgList, verbose bool) {
	if organizations == nil {
		fmt.Fprintf(writer, "No organizations found\n")
		return
	}

	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\n", "Name", "Status", "Description")
	}

	for _, organization := range *organizations {
		name := "N/A"
		if organization.Name != nil {
			name = *organization.Name
		}

		status := "Unknown"
		if organization.Status != nil && organization.Status.OrgStatus != nil && organization.Status.OrgStatus.StatusIndicator != nil {
			status = *organization.Status.OrgStatus.StatusIndicator
		}

		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\n", name, status)
		} else {
			description := "N/A"
			if organization.Spec != nil && organization.Spec.Description != nil {
				description = *organization.Spec.Description
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", name, status, description)
		}
	}
}

// Prints output details of OS Profiles
func printOrganization(writer io.Writer, name string, organization *tenancy.GetorgOrg) {
	if organization == nil {
		fmt.Fprintf(writer, "Organization %s not found\n", name)
		return
	}

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", name)

	description := "N/A"
	if organization.Spec != nil && organization.Spec.Description != nil {
		description = *organization.Spec.Description
	}
	_, _ = fmt.Fprintf(writer, "Description: \t%s\n", description)

	status := "Unknown"
	message := "N/A"
	uid := "N/A"

	if organization.Status != nil && organization.Status.OrgStatus != nil {
		if organization.Status.OrgStatus.StatusIndicator != nil {
			status = *organization.Status.OrgStatus.StatusIndicator
		}
		if organization.Status.OrgStatus.Message != nil {
			message = *organization.Status.OrgStatus.Message
		}
		if organization.Status.OrgStatus.UID != nil {
			uid = *organization.Status.OrgStatus.UID
		}
	}

	_, _ = fmt.Fprintf(writer, "Status: \t%s\n", status)
	_, _ = fmt.Fprintf(writer, "Status message: \t%s\n", message)
	_, _ = fmt.Fprintf(writer, "UID: \t%s\n\n", uid)
}

func getGetOrganizationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "organization <name> [flags]",
		Short:   "Get a organization",
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
	cmd.PersistentFlags().StringP("description", "d", viper.GetString("description"), "Optional flag used to provide a description to a cloud init config resource")
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
	ctx, organizationClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := organizationClient.GETV1OrgsOrgOrgWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting organizations"); !proceed {
		return err
	}

	printOrganization(writer, name, resp.JSON200)
	return writer.Flush()
}

// Lists all Cloud Init configurations - retrieves all configurations and displays selected information in tabular format
func runListOrganizationCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, organizationClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := organizationClient.LISTV1OrgsWithResponse(ctx, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		OrganizationHeader, "error getting organizations"); !proceed {
		return err
	}

	printOrganizations(writer, resp.JSON200, verbose)

	return writer.Flush()
}

// Creates Organization
func runCreateOrganizationCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	desc := name
	descFlag, _ := cmd.Flags().GetString("description")
	if descFlag != "" {
		desc = descFlag
	}

	ctx, organizationClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := organizationClient.PUTV1OrgsOrgOrgWithResponse(ctx, name, &tenancy.PUTV1OrgsOrgOrgParams{},
		tenancy.PUTV1OrgsOrgOrgJSONRequestBody{
			Description: &desc,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, "error while creating organization")
}

// Deletes Organization - checks if a organization already exists and then deletes it if it does
func runDeleteOrganizationCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	ctx, organizationClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := organizationClient.DELETEV1OrgsOrgOrgWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting organization %s", name))
}
