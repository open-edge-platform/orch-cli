// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

func (s *CLITestSuite) listGroups(args commandArgs) (string, error) {
	commandString := addCommandArgs(args, "list groups")
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestListGroups() {
	listOutput, err := s.listGroups(make(map[string]string))
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name": "org-admin-group",
			"ID":   "group-uuid-1",
		},
		{
			"Name": "edge-manager-group",
			"ID":   "group-uuid-2",
		},
		{
			"Name": "edge-operator-group",
			"ID":   "group-uuid-3",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)
}

func (s *CLITestSuite) TestListGroupsVerbose() {
	CArgs := map[string]string{
		"verbose": "true",
	}
	listOutput, err := s.listGroups(CArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name": "org-admin-group",
			"ID":   "group-uuid-1",
			"Path": "/org-admin-group",
		},
		{
			"Name": "edge-manager-group",
			"ID":   "group-uuid-2",
			"Path": "/edge-manager-group",
		},
		{
			"Name": "edge-operator-group",
			"ID":   "group-uuid-3",
			"Path": "/edge-operator-group",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)
}
