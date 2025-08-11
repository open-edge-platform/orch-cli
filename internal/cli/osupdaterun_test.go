// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) listOSUpdateRun(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list osupdaterun --project %s`,
		publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getOSUpdateRun(publisher string, id string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get osupdaterun %s --project %s`, id, publisher))
	return s.runCommand(commandString)
}
func (s *CLITestSuite) deleteOSUpdateRun(publisher string, id string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete osupdaterun %s --project %s`, id, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestOSUpdateRun() {
	id := "osupdaterun-abc12345"
	/////////////////////////////
	// Test OS Update Run List
	/////////////////////////////

	//List OS Update Runs
	OArgs := map[string]string{}
	_, err := s.listOSUpdateRun(project, OArgs)
	s.NoError(err)

	//List OS Update Runs --verbose
	OArgs = map[string]string{
		"verbose": "",
	}
	listOutput, err := s.listOSUpdateRun(project, OArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name":           "security-update-jan-2025",
			"Resource ID":    "osupdate-run-abc123",
			"Status":         "completed",
			"Applied Policy": "security-policy-v1.2",
			"Start Time":     "2025-01-15 10:30:00 +0000 UTC",
			"End Time":       "2025-01-15 10:30:00 +0000 UTC",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	/////////////////////////////
	// Test OS Update Run Get
	/////////////////////////////

	//Get OS Update Runs
	OArgs = map[string]string{}
	getOutput, err := s.getOSUpdateRun(project, id, OArgs)
	s.NoError(err)

	parsedGetOutput := mapGetOutput(getOutput)

	expectedOutput := map[string]string{

		"OS Profile Field": "Value",
		"Name:":            "security-update-jan-2025",
		"ResourceID:":      id,
		"Status:":          "completed",
		"Status Detail:":   "All updates applied successfully",
		"Applied Policy:":  "security-policy-v1.2",
		"Description:":     "Monthly security updates for edge devices",
		"Start Time:":      "2025-01-15 10:30:00 +0000 UTC",
		"End Time:":        "2025-01-15 10:30:00 +0000 UTC",
	}

	s.compareGetOutput(expectedOutput, parsedGetOutput)

	/////////////////////////////
	// Test OS Update Run Delete
	/////////////////////////////

	//Get OS Update Runs
	OArgs = map[string]string{}
	_, err = s.deleteOSUpdateRun(project, id, OArgs)
	s.NoError(err)
}
