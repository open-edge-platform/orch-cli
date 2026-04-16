// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) listUsers(args commandArgs) (string, error) {
	commandString := addCommandArgs(args, "list users")
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getUser(username string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get user "%s"`, username))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) createUser(username string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create user %s`, username))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteUser(username string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete user "%s"`, username))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) setUser(username string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`set user "%s"`, username))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestListUsers() {
	listOutput, err := s.listUsers(make(map[string]string))
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Username": "admin",
			"Enabled":  "true",
		},
		{
			"Username": "sample-user",
			"Enabled":  "true",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)
}

func (s *CLITestSuite) TestListUsersVerbose() {
	CArgs := map[string]string{
		"verbose": "true",
	}
	listOutput, err := s.listUsers(CArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Username":   "admin",
			"Email":      "admin@example.com",
			"First Name": "Admin",
			"Enabled":    "true",
		},
		{
			"Username":   "sample-user",
			"Email":      "sample-user@sample-domain.com",
			"First Name": "sample",
			"Enabled":    "true",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)
}

func (s *CLITestSuite) TestGetUser() {
	getOutput, err := s.getUser("sample-user", make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Username:":   "sample-user",
		"ID:":         "user-uuid-1234",
		"Email:":      "sample-user@sample-domain.com",
		"First Name:": "sample",
		"Last Name:":  "User",
		"Enabled:":    "true",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)
}

func (s *CLITestSuite) TestGetUserWithGroups() {
	CArgs := map[string]string{
		"groups": "true",
	}
	getOutput, err := s.getUser("sample-user", CArgs)
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Username:":   "sample-user",
		"ID:":         "user-uuid-1234",
		"Email:":      "sample-user@sample-domain.com",
		"First Name:": "sample",
		"Last Name:":  "User",
		"Enabled:":    "true",
		"Groups:":     "org-admin-group",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)
}

func (s *CLITestSuite) TestGetUserNotFound() {
	_, err := s.getUser("nonexistent-user", make(map[string]string))
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *CLITestSuite) TestCreateUser() {
	CArgs := map[string]string{
		"email":      "new@example.com",
		"first-name": "New",
		"last-name":  "User",
	}
	output, err := s.createUser("new-user", CArgs)
	s.NoError(err)
	s.Contains(output, "created successfully")
}

func (s *CLITestSuite) TestCreateUserWithInlinePassword() {
	CArgs := map[string]string{
		"password": "s3cret",
	}
	// Use "sample-user" because the mock's GetUserByUsername only
	// knows pre-seeded users (needed to look up the ID for SetPassword).
	output, err := s.createUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "created successfully")
}

func (s *CLITestSuite) TestCreateUserWithEnvPassword() {
	s.T().Setenv(passwordEnvVar, "env-s3cret")
	CArgs := map[string]string{
		"password": "",
	}
	output, err := s.createUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "created successfully")
}

func (s *CLITestSuite) TestSetUserPasswordPromptFailsWithoutTerminal() {
	// When --password is passed with no value and no env var is set,
	// resolvePassword falls through to the interactive prompt which
	// fails because tests don't run in a terminal.
	s.T().Setenv(passwordEnvVar, "")
	CArgs := map[string]string{
		"password": "",
	}
	_, err := s.setUser("sample-user", CArgs)
	s.Error(err)
}

func (s *CLITestSuite) TestSetUserWithInlinePassword() {
	CArgs := map[string]string{
		"password": "new-s3cret",
	}
	output, err := s.setUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "Password updated")
}

func (s *CLITestSuite) TestSetUserWithEnvPassword() {
	s.T().Setenv(passwordEnvVar, "env-s3cret")
	CArgs := map[string]string{
		"password": "",
	}
	output, err := s.setUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "Password updated")
}

func (s *CLITestSuite) TestDeleteUser() {
	output, err := s.deleteUser("sample-user", make(map[string]string))
	s.NoError(err)
	s.Contains(output, "deleted successfully")
}

func (s *CLITestSuite) TestDeleteUserNotFound() {
	_, err := s.deleteUser("nonexistent-user", make(map[string]string))
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *CLITestSuite) TestSetUserAddGroup() {
	CArgs := map[string]string{
		"add-group": "edge-manager-group",
	}
	output, err := s.setUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "Added user")
	s.Contains(output, "edge-manager-group")
}

func (s *CLITestSuite) TestSetUserRemoveGroup() {
	CArgs := map[string]string{
		"remove-group": "org-admin-group",
	}
	output, err := s.setUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "Removed user")
	s.Contains(output, "org-admin-group")
}

func (s *CLITestSuite) TestSetUserGroupNotFound() {
	CArgs := map[string]string{
		"add-group": "nonexistent-group",
	}
	_, err := s.setUser("sample-user", CArgs)
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *CLITestSuite) TestSetUserAddRealmRole() {
	CArgs := map[string]string{
		"add-realm-role": "org1_proj1_m",
	}
	output, err := s.setUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "Assigned realm role")
	s.Contains(output, "org1_proj1_m")
}

func (s *CLITestSuite) TestSetUserRemoveRealmRole() {
	CArgs := map[string]string{
		"remove-realm-role": "org1_proj1_m",
	}
	output, err := s.setUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "Removed realm role")
	s.Contains(output, "org1_proj1_m")
}

func (s *CLITestSuite) TestSetUserRealmRoleNotFound() {
	CArgs := map[string]string{
		"add-realm-role": "nonexistent-role",
	}
	_, err := s.setUser("sample-user", CArgs)
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *CLITestSuite) TestSetUserAddGroupAndRealmRole() {
	CArgs := map[string]string{
		"add-group":      "edge-manager-group",
		"add-realm-role": "org1_proj1_m",
	}
	output, err := s.setUser("sample-user", CArgs)
	s.NoError(err)
	s.Contains(output, "Added user")
	s.Contains(output, "edge-manager-group")
	s.Contains(output, "Assigned realm role")
	s.Contains(output, "org1_proj1_m")
}

func (s *CLITestSuite) TestGetUserWithRoles() {
	CArgs := map[string]string{
		"roles": "true",
	}
	getOutput, err := s.getUser("sample-user", CArgs)
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	s.Equal("sample-user", parsedOutput["Username:"])
	s.Contains(parsedOutput["Realm Roles:"], "org1_proj1_m")
}

func (s *CLITestSuite) TestGetUserWithGroupsAndRoles() {
	CArgs := map[string]string{
		"groups": "true",
		"roles":  "true",
	}
	getOutput, err := s.getUser("sample-user", CArgs)
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	s.Equal("sample-user", parsedOutput["Username:"])
	s.Contains(parsedOutput["Groups:"], "org-admin-group")
	s.Contains(parsedOutput["Realm Roles:"], "org1_proj1_m")
}
