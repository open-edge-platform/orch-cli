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

var ProjectHeader = fmt.Sprintf("\n%s\t%s", "Name", "Status")

// Prints projects in tabular format
func printProjects(writer io.Writer, projects *tenancy.ProjectProjectList, verbose bool) {
	if projects == nil {
		fmt.Fprintf(writer, "No projects found\n")
		return
	}

	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\n", "Name", "Status", "Description")
	}

	for _, project := range *projects {
		name := "N/A"
		if project.Name != nil {
			name = *project.Name
		}

		status := "Unknown"
		if project.Status != nil && project.Status.ProjectStatus != nil && project.Status.ProjectStatus.StatusIndicator != nil {
			status = *project.Status.ProjectStatus.StatusIndicator
		}

		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\n", name, status)
		} else {
			description := "N/A"
			if project.Spec != nil && project.Spec.Description != nil {
				description = *project.Spec.Description
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", name, status, description)
		}
	}
}

// Prints output details of projects
func printProject(writer io.Writer, name string, project *tenancy.GetprojectProject) {
	if project == nil {
		fmt.Fprintf(writer, "Project %s not found\n", name)
		return
	}

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", name)

	description := "N/A"
	if project.Spec != nil && project.Spec.Description != nil {
		description = *project.Spec.Description
	}
	_, _ = fmt.Fprintf(writer, "Description: \t%s\n", description)

	status := "Unknown"
	message := "N/A"
	uid := "N/A"

	if project.Status != nil && project.Status.ProjectStatus != nil {
		if project.Status.ProjectStatus.StatusIndicator != nil {
			status = *project.Status.ProjectStatus.StatusIndicator
		}
		if project.Status.ProjectStatus.Message != nil {
			message = *project.Status.ProjectStatus.Message
		}
		if project.Status.ProjectStatus.UID != nil {
			uid = *project.Status.ProjectStatus.UID
		}
	}

	_, _ = fmt.Fprintf(writer, "Status: \t%s\n", status)
	_, _ = fmt.Fprintf(writer, "Status message: \t%s\n", message)
	_, _ = fmt.Fprintf(writer, "UID: \t%s\n\n", uid)
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

// Gets specific Cloud Init configuration bu resource ID
func runGetProjectCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	writer, verbose := getOutputContext(cmd)
	ctx, projectClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := projectClient.GETV1ProjectsProjectProjectWithResponse(ctx, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}

	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		"", "error getting projects"); !proceed {
		return err
	}

	printProject(writer, name, resp.JSON200)
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

	desc := name
	descFlag, _ := cmd.Flags().GetString("description")
	if descFlag != "" {
		desc = descFlag
	}

	ctx, projectClient, _, err := TenancyFactory(cmd)
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
