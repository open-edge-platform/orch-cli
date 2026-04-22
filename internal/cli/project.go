// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
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
	DEFAULT_PROJECT_FORMAT         = "table{{none .Name}}\t{{.StatusIndicator}}"
	DEFAULT_PROJECT_INSPECT_FORMAT = `Name: {{none .Name}}
Description: {{none .Description}}
Status: {{none .StatusIndicator}}
Status Message: {{none .StatusMessage}}
UID: {{none .UID}}`
	PROJECT_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_PROJECT_OUTPUT_TEMPLATE"
)

const listProjectExamples = `# List all projects in the organization
orch-cli list projects
`

const getProjectExamples = `# Get detailed information about specific project
orch-cli get project myproject
`

const createProjectExamples = `# Create a project with a given name
orch-cli create project myproject

# Create a project with a given name and description
orch-cli create project myproject --description "my description"
`

const deleteProjectExamples = `#Delete a project using it's name
orch-cli delete project myproject`

// ProjectListItem is a flattened view for template output
type ProjectListItem struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	StatusIndicator *string `json:"statusIndicator,omitempty"`
	StatusMessage   *string `json:"statusMessage,omitempty"`
	UID             *string `json:"uid,omitempty"`
}

func flattenProjects(projects *tenancy.ProjectProjectList) []ProjectListItem {
	if projects == nil {
		return []ProjectListItem{}
	}

	items := make([]ProjectListItem, 0, len(*projects))
	for _, proj := range *projects {
		item := ProjectListItem{
			Name: proj.Name,
		}
		if proj.Spec != nil {
			item.Description = proj.Spec.Description
		}
		if proj.Status != nil && proj.Status.ProjectStatus != nil {
			item.StatusIndicator = proj.Status.ProjectStatus.StatusIndicator
			item.StatusMessage = proj.Status.ProjectStatus.Message
			item.UID = proj.Status.ProjectStatus.UID
		}
		items = append(items, item)
	}
	return items
}

func getProjectOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	if verbose {
		return DEFAULT_PROJECT_INSPECT_FORMAT, nil
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_PROJECT_FORMAT, PROJECT_OUTPUT_TEMPLATE_ENVVAR)
}

func printProjects(cmd *cobra.Command, writer io.Writer, projects *tenancy.ProjectProjectList, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getProjectOutputFormat(cmd, verbose)
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

	items := flattenProjects(projects)

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

func printProject(cmd *cobra.Command, writer io.Writer, name string, project *tenancy.GetprojectProject) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	item := ProjectListItem{
		Name: &name,
	}
	if project != nil {
		if project.Spec != nil {
			item.Description = project.Spec.Description
		}
		if project.Status != nil && project.Status.ProjectStatus != nil {
			item.StatusIndicator = project.Status.ProjectStatus.StatusIndicator
			item.StatusMessage = project.Status.ProjectStatus.Message
			item.UID = project.Status.ProjectStatus.UID
		}
	}

	outputFormat := DEFAULT_PROJECT_INSPECT_FORMAT

	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      []ProjectListItem{item},
	}

	GenerateOutput(writer, &result)
	return nil
}

func getGetProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project <name> [flags]",
		Short:   "Get a project",
		Example: getProjectExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: projectAliases,
		RunE:    runGetProjectCommand,
	}
	addStandardGetOutputFlags(cmd)
	return cmd
}

func getListProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project [flags]",
		Short:   "List all projects",
		Example: listProjectExamples,
		Aliases: projectAliases,
		RunE:    runListProjectCommand,
	}
	cmd.Flags().String("order-by", "", "order results by field (table output only)")
	addStandardListOutputFlags(cmd)
	return cmd
}

func getCreateProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project [flags]",
		Short:   "Creates a project",
		Example: createProjectExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: projectAliases,
		RunE:    runCreateProjectCommand,
	}
	cmd.PersistentFlags().StringP("description", "d", viper.GetString("description"), "Optional flag used to provide a description to a cloud init config resource")
	return cmd
}

func getDeleteProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project <name> [flags]",
		Short:   "Delete a project",
		Example: deleteProjectExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: projectAliases,
		RunE:    runDeleteProjectCommand,
	}
	return cmd
}

// Gets specific project by name
func runGetProjectCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	writer, _ := getOutputContext(cmd)
	ctx, projectClient, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := projectClient.GETV1ProjectsProjectProjectWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting projects"); !proceed {
		return err
	}

	if err := printProject(cmd, writer, name, resp.JSON200); err != nil {
		return err
	}
	return writer.Flush()
}

// Lists all projects
func runListProjectCommand(cmd *cobra.Command, _ []string) error {
	writer, _ := getOutputContext(cmd)

	ctx, projectClient, err := TenancyFactory(cmd)
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
		validatedOrderBy, err = normalizeOrderByForClientSorting(raw, ProjectListItem{})
	} else {
		// JSON/YAML: no API support, but allow any field for consistency
		if raw != "" {
			validatedOrderBy = &raw
		}
	}
	if err != nil {
		return err
	}

	resp, err := projectClient.LISTV1ProjectsWithResponse(ctx, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, true,
		"", "error getting projects"); !proceed {
		return err
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printProjects(cmd, writer, resp.JSON200, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}

	return writer.Flush()
}

// Creates Project
func runCreateProjectCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	desc := name
	descFlag, _ := cmd.Flags().GetString("description")
	if descFlag != "" {
		desc = descFlag
	}

	ctx, projectClient, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := projectClient.PUTV1ProjectsProjectProjectWithResponse(ctx, name, &tenancy.PUTV1ProjectsProjectProjectParams{},
		tenancy.PUTV1ProjectsProjectProjectJSONRequestBody{
			Description: &desc,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, "error while creating project")
}

// Deletes Project - checks if a project already exists and then deletes it if it does
func runDeleteProjectCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	ctx, projectClient, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := projectClient.DELETEV1ProjectsProjectProjectWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting project %s", name))
}
