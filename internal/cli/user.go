// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/rest/keycloak"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	// passwordPromptSentinel is the NoOptDefVal for --password, allowing
	// Cobra to accept the flag without a value (e.g. --password).
	// A non-empty string is required because Cobra treats NoOptDefVal=""
	// the same as unset, causing --password without =value to error.
	passwordPromptSentinel = "__prompt__"
	passwordEnvVar         = "ORCH_PASSWORD"

	DEFAULT_USER_FORMAT         = "table{{.Username}}\t{{.Enabled}}"
	DEFAULT_USER_VERBOSE_FORMAT = "table{{.Username}}\t{{none .Email}}\t{{none .FirstName}}\t{{.Enabled}}"
	DEFAULT_USER_INSPECT_FORMAT = `Username: {{.Username}}
ID: {{.ID}}
Email: {{none .Email}}
First Name: {{none .FirstName}}
Last Name: {{none .LastName}}
Enabled: {{.Enabled}}{{if .Groups}}
Groups: {{.Groups}}{{end}}{{if .RealmRoles}}
Realm Roles: {{.RealmRoles}}{{end}}`
	USER_OUTPUT_TEMPLATE_ENVVAR = "ORCH_CLI_USER_OUTPUT_TEMPLATE"
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

# Get user details including realm role assignments
orch-cli get user sample-user --roles

# Get user details with both groups and roles
orch-cli get user sample-user --groups --roles
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

# Assign a realm role to a user
orch-cli set user sample-user --add-realm-role "${ORG_UID}_${PROJ_UID}_m"

# Remove a realm role from a user
orch-cli set user sample-user --remove-realm-role "${ORG_UID}_${PROJ_UID}_m"

# Combine group and realm role changes
orch-cli set user sample-user --add-group edge-manager-group --add-realm-role "${ORG_UID}_${PROJ_UID}_m"
`

// UserListItem is a flattened view for template output
type UserListItem struct {
	Username   string  `json:"username,omitempty"`
	Email      *string `json:"email,omitempty"`
	FirstName  *string `json:"firstName,omitempty"`
	LastName   *string `json:"lastName,omitempty"`
	ID         string  `json:"id,omitempty"`
	Enabled    string  `json:"enabled,omitempty"`
	Groups     *string `json:"groups,omitempty"`
	RealmRoles *string `json:"realmRoles,omitempty"`
}

func flattenUsers(users []keycloak.UserRepresentation) []UserListItem {
	items := make([]UserListItem, 0, len(users))
	for _, user := range users {
		enabled := "true"
		if user.Enabled != nil && !*user.Enabled {
			enabled = "false"
		}
		item := UserListItem{
			Username: user.Username,
			ID:       user.ID,
			Enabled:  enabled,
		}
		if user.Email != "" {
			item.Email = &user.Email
		}
		if user.FirstName != "" {
			item.FirstName = &user.FirstName
		}
		if user.LastName != "" {
			item.LastName = &user.LastName
		}
		items = append(items, item)
	}
	return items
}

func flattenUser(user *keycloak.UserRepresentation, groups []keycloak.GroupRepresentation, roles []keycloak.RoleRepresentation) UserListItem {
	enabled := "true"
	if user.Enabled != nil && !*user.Enabled {
		enabled = "false"
	}

	item := UserListItem{
		Username: user.Username,
		ID:       user.ID,
		Enabled:  enabled,
	}

	if user.Email != "" {
		item.Email = &user.Email
	}
	if user.FirstName != "" {
		item.FirstName = &user.FirstName
	}
	if user.LastName != "" {
		item.LastName = &user.LastName
	}

	if groups != nil && len(groups) > 0 {
		groupNames := make([]string, 0, len(groups))
		for _, g := range groups {
			groupNames = append(groupNames, g.Name)
		}
		groupsStr := strings.Join(groupNames, ", ")
		item.Groups = &groupsStr
	}

	if roles != nil && len(roles) > 0 {
		roleNames := make([]string, 0, len(roles))
		for _, r := range roles {
			roleNames = append(roleNames, r.Name)
		}
		rolesStr := strings.Join(roleNames, ", ")
		item.RealmRoles = &rolesStr
	}

	return item
}

func getUserOutputFormat(cmd *cobra.Command, verbose bool) (string, error) {
	// Check if we're in get command context (has --groups or --roles flags)
	showGroups, _ := cmd.Flags().GetBool("groups")
	showRoles, _ := cmd.Flags().GetBool("roles")
	if showGroups || showRoles {
		return DEFAULT_USER_INSPECT_FORMAT, nil
	}

	if verbose {
		return DEFAULT_USER_VERBOSE_FORMAT, nil
	}
	return resolveTableOutputTemplate(cmd, DEFAULT_USER_FORMAT, USER_OUTPUT_TEMPLATE_ENVVAR)
}

func printUsers(cmd *cobra.Command, writer io.Writer, users []keycloak.UserRepresentation, orderBy *string, outputFilter *string, verbose bool) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	outputFormat, err := getUserOutputFormat(cmd, verbose)
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

	items := flattenUsers(users)

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

func printUser(cmd *cobra.Command, writer io.Writer, user *keycloak.UserRepresentation, groups []keycloak.GroupRepresentation, roles []keycloak.RoleRepresentation) error {
	outputType, _ := cmd.Flags().GetString("output-type")

	item := flattenUser(user, groups, roles)
	outputFormat := DEFAULT_USER_INSPECT_FORMAT

	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(outputType),
		NameLimit: -1,
		Data:      item,
	}

	GenerateOutput(writer, &result)
	return nil
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
	cmd.Flags().String("order-by", "", "order results by field (table output only)")
	addStandardListOutputFlags(cmd)
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
	cmd.Flags().Bool("roles", false, "Also list the user's realm role assignments")
	cmd.Flags().String("realm", "master", "Keycloak realm")
	addStandardGetOutputFlags(cmd)
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
		Short:   "Update a user (password, group membership, realm roles)",
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
	cmd.Flags().StringSlice("add-realm-role", nil, "Realm role name(s) to assign to the user")
	cmd.Flags().StringSlice("remove-realm-role", nil, "Realm role name(s) to remove from the user")
	cmd.Flags().String("realm", "master", "Keycloak realm")
	return cmd
}

func runListUsersCommand(cmd *cobra.Command, _ []string) error {
	writer, _ := getOutputContext(cmd)

	ctx, kcClient, realm, err := KeycloakAdminFactory(cmd)
	if err != nil {
		return err
	}

	raw, err := cmd.Flags().GetString("order-by")
	if err != nil {
		return err
	}

	outputType, _ := cmd.Flags().GetString("output-type")
	verbose, _ := cmd.Flags().GetBool("verbose")

	var validatedOrderBy *string
	if outputType == "table" {
		validatedOrderBy, err = normalizeOrderByForClientSorting(raw, UserListItem{})
	} else {
		// JSON/YAML: no API support, but allow any field for consistency
		if raw != "" {
			validatedOrderBy = &raw
		}
	}
	if err != nil {
		return err
	}

	users, err := kcClient.ListUsers(ctx, realm)
	if err != nil {
		return fmt.Errorf("error listing users: %w", err)
	}

	outputFilter, _ := cmd.Flags().GetString("output-filter")
	if err := printUsers(cmd, writer, users, validatedOrderBy, &outputFilter, verbose); err != nil {
		return err
	}

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

	var roles []keycloak.RoleRepresentation
	showRoles, _ := cmd.Flags().GetBool("roles")
	if showRoles {
		roles, err = kcClient.ListUserRealmRoles(ctx, realm, user.ID)
		if err != nil {
			return fmt.Errorf("error getting user realm roles: %w", err)
		}
	}

	if err := printUser(cmd, writer, user, groups, roles); err != nil {
		return err
	}
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

	addRoles, _ := cmd.Flags().GetStringSlice("add-realm-role")
	removeRoles, _ := cmd.Flags().GetStringSlice("remove-realm-role")

	for _, roleName := range addRoles {
		role, err := kcClient.GetRealmRoleByName(ctx, realm, roleName)
		if err != nil {
			return fmt.Errorf("error looking up realm role %q: %w", roleName, err)
		}
		if err := kcClient.AddRealmRolesToUser(ctx, realm, user.ID, []keycloak.RoleRepresentation{*role}); err != nil {
			return fmt.Errorf("error assigning realm role %q: %w", roleName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Assigned realm role %q to user %q\n", roleName, username)
	}

	for _, roleName := range removeRoles {
		role, err := kcClient.GetRealmRoleByName(ctx, realm, roleName)
		if err != nil {
			return fmt.Errorf("error looking up realm role %q: %w", roleName, err)
		}
		if err := kcClient.RemoveRealmRolesFromUser(ctx, realm, user.ID, []keycloak.RoleRepresentation{*role}); err != nil {
			return fmt.Errorf("error removing realm role %q: %w", roleName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed realm role %q from user %q\n", roleName, username)
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
