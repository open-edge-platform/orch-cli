// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"

	"github.com/Nerzal/gocloak/v13"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/tenancy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listUserExamples = `# List all users in the organization
orch-cli list users
`

const getUserExamples = `# Get detailed information about specific user
orch-cli get user myuser
`

const createUserExamples = `# Create a user with a given name using cloud init file as input
orch-cli create user myuser
`

const deleteUserExamples = `#Delete a user using it's name
orch-cli delete user myuser`

var UserHeader = fmt.Sprintf("\n%s\t%s", "Name", "Status")

// Prints OS Profiles in tabular format
func printUsers(writer io.Writer, users []*gocloak.User, verbose bool) {
	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\n", "Name", "Status", "Description")
	}

	for _, user := range users {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\n", *user.Username, *user.Enabled)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", *user.Username, *user.Enabled, *user.ID)
		}
	}
}

// Prints output details of OS Profiles
func printUser(writer io.Writer, name string, project *tenancy.GetprojectProject) {

	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", name)
	_, _ = fmt.Fprintf(writer, "Description: \t%s\n", *project.Spec.Description)
	_, _ = fmt.Fprintf(writer, "Status: \t%s\n", *project.Status.ProjectStatus.StatusIndicator)
	_, _ = fmt.Fprintf(writer, "Status message: \t%s\n", *project.Status.ProjectStatus.Message)
	_, _ = fmt.Fprintf(writer, "UID: \t%s\n\n", *project.Status.ProjectStatus.UID)

}

func getGetUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user <name> [flags]",
		Short:   "Get a user",
		Example: getUserExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: userAliases,
		RunE:    runGetUserCommand,
	}
	return cmd
}

func getListUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user [flags]",
		Short:   "List all users",
		Example: listUserExamples,
		Aliases: userAliases,
		RunE:    runListUserCommand,
	}
	return cmd
}

func getCreateUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user [flags]",
		Short:   "Creates a user",
		Example: createUserExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: userAliases,
		RunE:    runCreateUserCommand,
	}
	cmd.PersistentFlags().StringP("description", "d", viper.GetString("description"), "Optional flag used to provide a description to a cloud init config resource")
	return cmd
}

func getDeleteUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user <name> [flags]",
		Short:   "Delete a user",
		Example: deleteUserExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: userAliases,
		RunE:    runDeleteUserCommand,
	}
	return cmd
}

// Gets specific Cloud Init configuration bu resource ID
func runGetUserCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	writer, verbose := getOutputContext(cmd)
	ctx, userClient, _, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := userClient.GETV1ProjectsProjectProjectWithResponse(ctx, name, auth.AddAuthHeader)
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
func runListUserCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, userClient, _, err := getKCServiceContext(cmd)
	if err != nil {
		return err
	}
	params := gocloak.GetUsersParams{}
	token, err := auth.GetAccessToken(ctx)
	if err != nil {
		return err
	}
	users, err := userClient.GetUsers(ctx, token, "master", params)
	if err != nil {
		return processError(err)
	}

	printUsers(writer, users, verbose)

	return writer.Flush()
}

// Creates Project
func runCreateUserCommand(cmd *cobra.Command, args []string) error {
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
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating projet"))
}

// Deletes Project - checks if a project already exists and then deletes it if it does
func runDeleteUserCommand(cmd *cobra.Command, args []string) error {

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
