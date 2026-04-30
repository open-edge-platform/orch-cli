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
			"NAME": "org-admin-group",
			"ID":   "group-uuid-1",
		},
		{
			"NAME": "edge-manager-group",
			"ID":   "group-uuid-2",
		},
		{
			"NAME": "edge-operator-group",
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
			"NAME": "org-admin-group",
			"ID":   "group-uuid-1",
			"PATH": "/org-admin-group",
		},
		{
			"NAME": "edge-manager-group",
			"ID":   "group-uuid-2",
			"PATH": "/edge-manager-group",
		},
		{
			"NAME": "edge-operator-group",
			"ID":   "group-uuid-3",
			"PATH": "/edge-operator-group",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)
}
