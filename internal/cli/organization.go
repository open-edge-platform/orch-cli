// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/tenancy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DEFAULT_ORGANIZATION_FORMAT         = "table{{none .Name}}\t{{.StatusIndicator}}"
	DEFAULT_ORGANIZATION_VERBOSE_FORMAT = "table{{none .Name}}\t{{.StatusIndicator}}\t{{none .Description}}"
	DEFAULT_ORGANIZATION_INSPECT_FORMAT = "Name: \t{{none .Name}}\nDescription: \t{{none .Description}}\nStatus: \t{{none .StatusIndicator}}\nStatus Message: \t{{none .StatusMessage}}\nUID: \t{{none .UID}}"
	ORGANIZATION_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_ORGANIZATION_OUTPUT_TEMPLATE"
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

// OrganizationListItem is a flattened view for template output
type OrganizationListItem struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	StatusIndicator *string `json:"statusIndicator,omitempty"`
	StatusMessage   *string `json:"statusMessage,omitempty"`
	UID             *string `json:"uid,omitempty"`
}

func flattenOrganizations(organizations *tenancy.OrgOrgList) []OrganizationListItem {
	if organizations == nil {
		return []OrganizationListItem{}
	}

	items := make([]OrganizationListItem, 0, len(*organizations))
	for _, org := range *organizations {
		item := OrganizationListItem{
			Name: org.Name,
		}
		if org.Spec != nil {
			item.Description = org.Spec.Description
		}
		if org.Status != nil && org.Status.OrgStatus != nil {
			item.StatusIndicator = org.Status.OrgStatus.StatusIndicator
			item.StatusMessage = org.Status.OrgStatus.Message
			item.UID = org.Status.OrgStatus.UID
		}
		items = append(items, item)
	}
	return items
}

func getOrganizationOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_ORGANIZATION_VERBOSE_FORMAT, nil
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_ORGANIZATION_FORMAT, ORGANIZATION_OUTPUT_TEMPLATE_ENVVAR)
}

func printOrganizations(cmd *cobra.Command, writer io.Writer, organizations *tenancy.OrgOrgList, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getOrganizationOutputFormat(cmd, verbose)
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

	items := flattenOrganizations(organizations)

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    filterSpec,
		OrderBy:   sortSpec,
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      items,
	}

	GenerateOutput(writer, &result)
	return nil
}

func printOrganization(cmd *cobra.Command, writer io.Writer, name string, organization *tenancy.GetorgOrg) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	item := OrganizationListItem{
		Name: &name,
	}
	if organization != nil {
		if organization.Spec != nil {
			item.Description = organization.Spec.Description
		}
		if organization.Status != nil && organization.Status.OrgStatus != nil {
			item.StatusIndicator = organization.Status.OrgStatus.StatusIndicator
			item.StatusMessage = organization.Status.OrgStatus.Message
			item.UID = organization.Status.OrgStatus.UID
		}
	}

	outputFormat := DEFAULT_ORGANIZATION_INSPECT_FORMAT

	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      item,
	}

	GenerateOutput(writer, &result)
	return nil
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
	addStandardGetOutputFlags(cmd)
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
	cmd.Flags().String("order-by", "", "order results by field (table output only)")
	addStandardListOutputFlags(cmd)
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

// Gets specific organization by name
func runGetOrganizationCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	writer, _ := getOutputContext(cmd)
	ctx, organizationClient, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := organizationClient.GETV1OrgsOrgOrgWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting organizations"); !proceed {
		return err
	}

	if err := printOrganization(cmd, writer, name, resp.JSON200); err != nil {
		return err
	}
	return writer.Flush()
}

// Lists all organizations
func runListOrganizationCommand(cmd *cobra.Command, _ []string) error {
	writer, _ := getOutputContext(cmd)

	ctx, organizationClient, err := TenancyFactory(cmd)
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
		validatedOrderBy, err = normalizeOrderByForClientSorting(raw, OrganizationListItem{})
	} else {
		// JSON/YAML: no API support, but allow any field for consistency
		if raw != "" {
			validatedOrderBy = &raw
		}
	}
	if err != nil {
		return err
	}

	resp, err := organizationClient.LISTV1OrgsWithResponse(ctx, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting organizations"); !proceed {
		return err
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printOrganizations(cmd, writer, resp.JSON200, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}

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

	ctx, organizationClient, err := TenancyFactory(cmd)
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
	ctx, organizationClient, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := organizationClient.DELETEV1OrgsOrgOrgWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting organization %s", name))
}
