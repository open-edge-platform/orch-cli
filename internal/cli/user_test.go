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
