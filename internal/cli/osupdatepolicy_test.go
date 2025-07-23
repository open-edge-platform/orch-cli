// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) listOSUpdatePolicy(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list osupdatepolicy --project %s`,
		publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getOSUpdatePolicy(publisher string, id string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get osupdatepolicy %s --project %s`, id, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteOSUpdatePolicy(publisher string, id string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete osupdatepolicy %s --project %s`, id, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) createOSUpdatePolicy(publisher string, path string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create osupdatepolicy %s --project %s`, path, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestOSUpdatePolicy() {
	id := "osupdatepolicy-abc12345"
	path := "./testdata/latestosupdateprofile.yaml"

	/////////////////////////////
	// Test OS Update Policy Create
	/////////////////////////////

	//Create OS Update Policy immutable
	OArgs := map[string]string{}
	_, err := s.createOSUpdatePolicy(project, path, OArgs)
	s.NoError(err)

	//Create OS Update Policy mutable
	OArgs = map[string]string{}
	_, err = s.createOSUpdatePolicy(project, "./testdata/mutableosupdateprofile.yaml", OArgs)
	s.NoError(err)

	/////////////////////////////
	// Test OS Update Policy List
	/////////////////////////////

	//List OS Update Policy
	OArgs = map[string]string{}
	_, err = s.listOSUpdatePolicy(project, OArgs)
	s.NoError(err)

	//List OS Update Policy --verbose
	OArgs = map[string]string{
		"verbose": "",
	}
	listOutput, err := s.listOSUpdatePolicy(project, OArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name":         "security-policy-v1.2",
			"Resource ID":  "osupdatepolicy-abc12345",
			"Target OS ID": "os-1234abcd",
			"Description":  "Monthly security update policy",
			"Created":      "2025-01-15 10:30:00 +0000 UTC",
			"Updated":      "2025-01-15 10:30:00 +0000 UTC",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	/////////////////////////////
	// Test OS Update Policy Get
	/////////////////////////////

	//Get OS Update Policy
	OArgs = map[string]string{}
	getOutput, err := s.getOSUpdatePolicy(project, id, OArgs)
	s.NoError(err)

	parsedGetOutput := mapGetOutput(getOutput)

	expectedOutput := map[string]string{
		"Name:":             "security-policy-v1.2",
		"Resource ID:":      id, // "osupdatepolicy-abc12345"
		"Target OS ID:":     "os-1234abcd",
		"Description:":      "Monthly security update policy",
		"Install Packages:": "curl wget vim",
		"Update Policy:":    "UPDATE_POLICY_LATEST",
		"Create at:":        "2025-01-15 10:30:00 +0000 UTC",
		"Updated at:":       "2025-01-15 10:30:00 +0000 UTC",
	}
	s.compareGetOutput(expectedOutput, parsedGetOutput)

	/////////////////////////////
	// Test OS Update Policy Delete
	/////////////////////////////

	//Get OS Update Policy
	OArgs = map[string]string{}
	_, err = s.deleteOSUpdatePolicy(project, id, OArgs)
	s.NoError(err)
}
