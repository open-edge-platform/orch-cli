// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"time"

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
		fmt.Fprintf(writer, "\n%s\t%s\t%s\n", "Name", "Status", "User ID")
	}

	for _, user := range users {
		if !verbose {
			fmt.Fprintf(writer, "%s\t%v\n", *user.Username, *user.Enabled)
		} else {
			fmt.Fprintf(writer, "%s\t%v\t%s\n", *user.Username, *user.Enabled, *user.ID)
		}
	}
}

// Prints output details of OS Profiles
func printUser(writer io.Writer, user *gocloak.User, groups []*gocloak.Group) {

	email, id, fname, lname, enabled, emailVerified, createdAt := "", "", "", "", false, false, int64(0)
	if user.Email != nil {
		email = *user.Email
	}
	if user.ID != nil {
		id = *user.ID
	}
	if user.FirstName != nil {
		fname = *user.FirstName
	}
	if user.LastName != nil {
		lname = *user.LastName
	}
	if user.Enabled != nil {
		enabled = *user.Enabled
	}
	if user.EmailVerified != nil {
		emailVerified = *user.EmailVerified
	}
	if user.CreatedTimestamp != nil {
		createdAt = *user.CreatedTimestamp
	}
	_, _ = fmt.Fprintf(writer, "Name: \t%s\n", *user.Username)
	_, _ = fmt.Fprintf(writer, "Email: \t%s\n", email)
	_, _ = fmt.Fprintf(writer, "ID: \t%s\n", id)
	_, _ = fmt.Fprintf(writer, "First Name: \t%s\n", fname)
	_, _ = fmt.Fprintf(writer, "Last Name: \t%s\n", lname)
	_, _ = fmt.Fprintf(writer, "Status (Enabled): \t%v\n", enabled)
	_, _ = fmt.Fprintf(writer, "Email verified?: \t%v\n", emailVerified)
	// Convert Unix timestamp to readable time format
	if user.CreatedTimestamp != nil {
		createdTime := time.Unix(0, createdAt*int64(time.Millisecond))
		_, _ = fmt.Fprintf(writer, "Created at: \t%s\n", createdTime.Format("2006-01-02 15:04:05 MST"))
	} else {
		_, _ = fmt.Fprintf(writer, "Created at: \t%s\n", "N/A")
	}

	// Print each group in human-readable format
	if groups != nil && len(groups) > 0 {
		_, _ = fmt.Fprintf(writer, "Groups:\n")
		for i, group := range groups {
			groupName := "Unknown"
			if group.Name != nil {
				groupName = *group.Name
			}
			_, _ = fmt.Fprintf(writer, "  [%d]: \t%s\n", i+1, groupName)
		}
	} else {
		_, _ = fmt.Fprintf(writer, "Groups: \t%s\n", "None")
	}
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

// Gets specific user
func runGetUserCommand(cmd *cobra.Command, args []string) error {

	name := args[0]
	writer, _ := getOutputContext(cmd)
	ctx, userClient, _, err := getKCServiceContext(cmd)
	if err != nil {
		return err
	}
	params := gocloak.GetUsersParams{
		Username: &name,
	}
	token, err := auth.GetAccessToken(ctx)
	if err != nil {
		return err
	}
	users, err := userClient.GetUsers(ctx, token, "master", params)
	if err != nil {
		return processError(err)
	}

	//check precise username
	if len(users) == 0 || *users[0].Username != name {
		return fmt.Errorf("user %s not found", name)
	}

	gparams := gocloak.GetGroupsParams{}

	groups, err := userClient.GetUserGroups(ctx, token, "master", *users[0].ID, gparams)
	if err != nil {
		return processError(err)
	}

	printUser(writer, users[0], groups)
	return writer.Flush()
}

// Lists all users
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

// Creates User
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

// Deletes User - checks if a user already exists and then deletes it if it does
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
