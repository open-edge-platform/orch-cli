// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/open-edge-platform/cli/pkg/rest/keycloak"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	passwordPromptSentinel = "__prompt__"
	passwordEnvVar         = "ORCH_PASSWORD"
)

const listUsersExamples = `# List all users
orch-cli list users

# List all users in a specific realm
orch-cli list users --realm master
`

const getUserExamples = `# Get detailed information about a user
orch-cli get user sample-user

# Get user details including group memberships
orch-cli get user sample-user --groups
`

const createUserExamples = `# Create a user with just a username
orch-cli create user sample-user

# Create a user with all details
orch-cli create user sample-user --email sample@example.com --first-name Sample --last-name User

# Create a user and set a password (will prompt interactively)
orch-cli create user sample-user --password

# Create a user with an inline password (caution: visible in shell history)
orch-cli create user sample-user --password="s3cret"

# Create a user with password from environment variable
ORCH_PASSWORD=s3cret orch-cli create user sample-user --password
`

const deleteUserExamples = `# Delete a user by username
orch-cli delete user sample-user
`

const setUserExamples = `# Set a user's password (will prompt interactively)
orch-cli set user sample-user --password

# Set a user's password inline (caution: visible in shell history)
orch-cli set user sample-user --password="s3cret"

# Set a user's password from environment variable
ORCH_PASSWORD=s3cret orch-cli set user sample-user --password

# Add a user to a group
orch-cli set user sample-user --add-group org-admin-group

# Remove a user from a group
orch-cli set user sample-user --remove-group org-admin-group

# Add and remove groups in one command
orch-cli set user sample-user --add-group edge-manager-group --remove-group edge-operator-group
`

func printUsers(writer io.Writer, users []keycloak.UserRepresentation, verbose bool) {
	if len(users) == 0 {
		fmt.Fprintf(writer, "No users found\n")
		return
	}

	if verbose {
		fmt.Fprintf(writer, "\n%s\t%s\t%s\t%s\n", "Username", "Email", "First Name", "Enabled")
	}

	for _, user := range users {
		enabled := "true"
		if user.Enabled != nil && !*user.Enabled {
			enabled = "false"
		}

		if !verbose {
			fmt.Fprintf(writer, "%s\t%s\n", user.Username, enabled)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", user.Username, user.Email, user.FirstName, enabled)
		}
	}
}

func printUser(writer io.Writer, user *keycloak.UserRepresentation, groups []keycloak.GroupRepresentation) {
	if user == nil {
		fmt.Fprintf(writer, "User not found\n")
		return
	}

	_, _ = fmt.Fprintf(writer, "Username: \t%s\n", user.Username)
	_, _ = fmt.Fprintf(writer, "ID: \t%s\n", user.ID)
	_, _ = fmt.Fprintf(writer, "Email: \t%s\n", valueOrDefault(user.Email))
	_, _ = fmt.Fprintf(writer, "First Name: \t%s\n", valueOrDefault(user.FirstName))
	_, _ = fmt.Fprintf(writer, "Last Name: \t%s\n", valueOrDefault(user.LastName))

	enabled := "true"
	if user.Enabled != nil && !*user.Enabled {
		enabled = "false"
	}
	_, _ = fmt.Fprintf(writer, "Enabled: \t%s\n", enabled)

	if groups != nil {
		groupNames := make([]string, 0, len(groups))
		for _, g := range groups {
			groupNames = append(groupNames, g.Name)
		}
		_, _ = fmt.Fprintf(writer, "Groups: \t%s\n", strings.Join(groupNames, ", "))
	}
}

func valueOrDefault(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func getListUsersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users [flags]",
		Short:   "List all users",
		Example: listUsersExamples,
		Aliases: userAliases,
		RunE:    runListUsersCommand,
	}
	cmd.Flags().String("realm", "master", "Keycloak realm")
	return cmd
}

func getGetUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user <username> [flags]",
		Short:   "Get a user",
		Example: getUserExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: userAliases,
		RunE:    runGetUserCommand,
	}
	cmd.Flags().Bool("groups", false, "Also list the user's group memberships")
	cmd.Flags().String("realm", "master", "Keycloak realm")
	return cmd
}

func getCreateUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user <username> [flags]",
		Short:   "Create a user",
		Example: createUserExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: userAliases,
		RunE:    runCreateUserCommand,
	}
	cmd.Flags().String("email", "", "User email address")
	cmd.Flags().String("first-name", "", "User first name")
	cmd.Flags().String("last-name", "", "User last name")
	cmd.Flags().Bool("disabled", false, "Create user in disabled state")
	cmd.Flags().String("password", "", "Set password for the user (prompts if no value given, also reads ORCH_PASSWORD env var)")
	cmd.Flag("password").NoOptDefVal = passwordPromptSentinel
	cmd.Flags().Bool("temporary-password", false, "If set, user must change password on first login")
	cmd.Flags().String("realm", "master", "Keycloak realm")
	return cmd
}

func getDeleteUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user <username> [flags]",
		Short:   "Delete a user",
		Example: deleteUserExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: userAliases,
		RunE:    runDeleteUserCommand,
	}
	cmd.Flags().String("realm", "master", "Keycloak realm")
	return cmd
}

func getSetUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user <username> [flags]",
		Short:   "Update a user (password, group membership)",
		Example: setUserExamples,
		Args:    cobra.ExactArgs(1),
		Aliases: userAliases,
		RunE:    runSetUserCommand,
	}
	cmd.Flags().String("password", "", "Set password for the user (prompts if no value given, also reads ORCH_PASSWORD env var)")
	cmd.Flag("password").NoOptDefVal = passwordPromptSentinel
	cmd.Flags().Bool("temporary-password", false, "If set, user must change password on first login")
	cmd.Flags().StringSlice("add-group", nil, "Group name(s) to add the user to")
	cmd.Flags().StringSlice("remove-group", nil, "Group name(s) to remove the user from")
	cmd.Flags().String("realm", "master", "Keycloak realm")
	return cmd
}

func runListUsersCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)

	ctx, kcClient, realm, err := KeycloakAdminFactory(cmd)
	if err != nil {
		return err
	}

	users, err := kcClient.ListUsers(ctx, realm)
	if err != nil {
		return fmt.Errorf("error listing users: %w", err)
	}

	if !verbose {
		_, _ = fmt.Fprintf(writer, "\n%s\t%s\n", "Username", "Enabled")
	}

	printUsers(writer, users, verbose)
	return writer.Flush()
}

func runGetUserCommand(cmd *cobra.Command, args []string) error {
	username := args[0]
	writer, _ := getOutputContext(cmd)

	ctx, kcClient, realm, err := KeycloakAdminFactory(cmd)
	if err != nil {
		return err
	}

	user, err := kcClient.GetUserByUsername(ctx, realm, username)
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
	}

	var groups []keycloak.GroupRepresentation
	showGroups, _ := cmd.Flags().GetBool("groups")
	if showGroups {
		groups, err = kcClient.ListUserGroups(ctx, realm, user.ID)
		if err != nil {
			return fmt.Errorf("error getting user groups: %w", err)
		}
	}

	printUser(writer, user, groups)
	return writer.Flush()
}

func runCreateUserCommand(cmd *cobra.Command, args []string) error {
	username := args[0]

	ctx, kcClient, realm, err := KeycloakAdminFactory(cmd)
	if err != nil {
		return err
	}

	email, _ := cmd.Flags().GetString("email")
	firstName, _ := cmd.Flags().GetString("first-name")
	lastName, _ := cmd.Flags().GetString("last-name")
	disabled, _ := cmd.Flags().GetBool("disabled")

	enabled := !disabled
	user := keycloak.UserRepresentation{
		Username:  username,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Enabled:   &enabled,
	}

	if err := kcClient.CreateUser(ctx, realm, user); err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}

	if cmd.Flags().Changed("password") {
		password, err := resolvePassword(cmd)
		if err != nil {
			return err
		}

		createdUser, err := kcClient.GetUserByUsername(ctx, realm, username)
		if err != nil {
			return fmt.Errorf("user created but failed to look up for password set: %w", err)
		}

		temporary, _ := cmd.Flags().GetBool("temporary-password")
		if err := kcClient.SetPassword(ctx, realm, createdUser.ID, password, temporary); err != nil {
			return fmt.Errorf("user created but failed to set password: %w", err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "User %q created successfully\n", username)
	return nil
}

func runDeleteUserCommand(cmd *cobra.Command, args []string) error {
	username := args[0]

	ctx, kcClient, realm, err := KeycloakAdminFactory(cmd)
	if err != nil {
		return err
	}

	user, err := kcClient.GetUserByUsername(ctx, realm, username)
	if err != nil {
		return fmt.Errorf("error finding user: %w", err)
	}

	if err := kcClient.DeleteUser(ctx, realm, user.ID); err != nil {
		return fmt.Errorf("error deleting user %s: %w", username, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "User %q deleted successfully\n", username)
	return nil
}

func runSetUserCommand(cmd *cobra.Command, args []string) error {
	username := args[0]

	ctx, kcClient, realm, err := KeycloakAdminFactory(cmd)
	if err != nil {
		return err
	}

	user, err := kcClient.GetUserByUsername(ctx, realm, username)
	if err != nil {
		return fmt.Errorf("error finding user: %w", err)
	}

	// Handle password change
	if cmd.Flags().Changed("password") {
		password, err := resolvePassword(cmd)
		if err != nil {
			return err
		}
		temporary, _ := cmd.Flags().GetBool("temporary-password")
		if err := kcClient.SetPassword(ctx, realm, user.ID, password, temporary); err != nil {
			return fmt.Errorf("error setting password: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Password updated for user %q\n", username)
	}

	// Handle add-group
	addGroups, _ := cmd.Flags().GetStringSlice("add-group")
	removeGroups, _ := cmd.Flags().GetStringSlice("remove-group")

	if len(addGroups) > 0 || len(removeGroups) > 0 {
		allGroups, err := kcClient.ListGroups(ctx, realm)
		if err != nil {
			return fmt.Errorf("error listing groups: %w", err)
		}

		groupMap := make(map[string]string, len(allGroups))
		for _, g := range allGroups {
			groupMap[g.Name] = g.ID
		}

		for _, groupName := range addGroups {
			groupID, ok := groupMap[groupName]
			if !ok {
				return fmt.Errorf("group %q not found", groupName)
			}
			if err := kcClient.AddUserToGroup(ctx, realm, user.ID, groupID); err != nil {
				return fmt.Errorf("error adding user to group %q: %w", groupName, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added user %q to group %q\n", username, groupName)
		}

		for _, groupName := range removeGroups {
			groupID, ok := groupMap[groupName]
			if !ok {
				return fmt.Errorf("group %q not found", groupName)
			}
			if err := kcClient.RemoveUserFromGroup(ctx, realm, user.ID, groupID); err != nil {
				return fmt.Errorf("error removing user from group %q: %w", groupName, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed user %q from group %q\n", username, groupName)
		}
	}

	return nil
}

// resolvePassword determines the password from three sources in priority order:
//  1. Inline flag value (--password="value")
//  2. Environment variable (ORCH_PASSWORD)
//  3. Interactive terminal prompt
func resolvePassword(cmd *cobra.Command) (string, error) {
	flagVal, _ := cmd.Flags().GetString("password")

	// If an explicit value was provided inline (not the sentinel), use it
	if flagVal != passwordPromptSentinel && flagVal != "" {
		return flagVal, nil
	}

	// Check environment variable
	if envVal := os.Getenv(passwordEnvVar); envVal != "" {
		return envVal, nil
	}

	// Fall back to interactive prompt
	fmt.Fprint(cmd.OutOrStdout(), "Enter Password: ")
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	if len(bytePassword) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}
	password := string(bytePassword)
	// Zero the byte slice to reduce the window where the password is in memory
	for i := range bytePassword {
		bytePassword[i] = 0
	}
	return password, nil
}
