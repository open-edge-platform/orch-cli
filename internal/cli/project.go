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

const listProjectExamples = `# List all projects in the organization
orch-cli list projects
`

const getProjectExamples = `# Get detailed information about specific project
orch-cli get project myproject
`

const createProjectExamples = `# Create a project with a given name using cloud init file as input
orch-cli create project myproject`

const deleteProjectExamples = `#Delete a project using it's name
orch-cli delete project myproject`

var ProjectHeader = fmt.Sprintf("\n%s\t%s\t%s", "Name", "Resource ID", "Description")

// Prints OS Profiles in tabular format
func printProjects(writer io.Writer, projects *tenancy.ProjectProjectList, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\t%s\n", "Name", "Resource ID", "Description", "Creation Timestamp", "Updated Timestamp")
	}
	// for _, project := range projects {
	// 	if !verbose {
	// 		fmt.Fprintf(writer, "%s\t%s\t%s\n", project, *project.ResourceId, *project.Description)
	// 	} else {

	// 		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", project.Name, *project.ResourceId, *project.Description, project.Timestamps.CreatedAt, project.Timestamps.UpdatedAt)
	// 	}
	// }
}

// Prints output details of OS Profiles
func printProject(writer io.Writer, project *tenancy.GetprojectProject) {

	// _, _ = fmt.Fprintf(writer, "Name: \t%s\n", project.Name)
	// _, _ = fmt.Fprintf(writer, "Resource ID: \t%s\n", *project.ResourceId)
	// _, _ = fmt.Fprintf(writer, "Description: \t%s\n\n", *project.Description)
	// _, _ = fmt.Fprintf(writer, "Cloud Init:\n%s\n", project.Config)
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

func getGetProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project <name> [flags]",
		Short:   "Get a project",
		Example: getProjectExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: projectAliases,
		RunE:    runGetProjectCommand,
	}
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
	//cmd.PersistentFlags().StringP("filter", "f", viper.GetString("filter"), "Optional filter provided as part of cloud init list command\nUsage:\n\tCustom filter: --filter \"<custom filter>\" ie. --filter <filter> see https://google.aip.dev/160 and API spec.")
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
	//cmd.PersistentFlags().StringP("description", "d", viper.GetString("description"), "Optional flag used to provide a description to a cloud init config resource")
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

// Gets specific Cloud Init configuration bu resource ID
func runGetProjectCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	writer, verbose := getOutputContext(cmd)
	ctx, projectClient, _, err := TenancyFactory(cmd)
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

	resp, err := projectClient.GETV1ProjectsProjectProjectWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting projects"); !proceed {
		return err
	}

	printProject(writer, resp.JSON200)
	return writer.Flush()
}

// Lists all Cloud Init configurations - retrieves all configurations and displays selected information in tabular format
func runListProjectCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, projectClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := projectClient.LISTV1ProjectsWithResponse(ctx, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		ProjectHeader, "error getting projects"); !proceed {
		return err
	}

	printProjects(writer, resp.JSON200, verbose)

	return writer.Flush()
}

// Creates Project
func runCreateProjectCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	var desc *string
	descFlag, _ := cmd.Flags().GetString("description")
	if descFlag != "" {
		desc = &descFlag
	}

	ctx, projectClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := projectClient.PUTV1ProjectsProjectProjectWithResponse(ctx, name, &tenancy.PUTV1ProjectsProjectProjectParams{},
		tenancy.PUTV1ProjectsProjectProjectJSONRequestBody{
			Description: desc,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating projet"))
}

// Deletes Project - checks if a project already exists and then deletes it if it does
func runDeleteProjectCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	ctx, projectClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := projectClient.DELETEV1ProjectsProjectProjectWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting project %s", name))
}
