// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"
	"testing"
)

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

func FuzzCreateOSUpdatePolicy(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "./testdata/mutableosupdateprofile.yaml")         // valid
	f.Add("project", "./testdata/latestosupdateprofile.yaml")          // valid
	f.Add("project", "")                                               // missing file
	f.Add("project", "./testdata/invalid.yaml")                        // invalid file
	f.Add("invalid-project", "./testdata/mutableosupdateprofile.yaml") // invalid project
	f.Add("project", "./testdata/duplicate.yaml")                      // duplicate name

	f.Fuzz(func(t *testing.T, project, path string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{}

		_, err := testSuite.createOSUpdatePolicy(project, path, args)

		// Error expectations
		if path != "./testdata/mutableosupdateprofile.yaml" && path != "./testdata/latestosupdateprofile.yaml" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing or invalid file, got: %v", err)
			}
			return
		}
		if project == "" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing project, got: %v", err)
			}
			return
		}
		if project == "invalid-project" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for invalid project, got: %v", err)
			}
			return
		}
		if strings.Contains(path, "duplicate") {
			if err == nil || !strings.Contains(err.Error(), "already exists") {
				t.Errorf("Expected error for duplicate OS Update Policy name, got: %v", err)
			}
			return
		}
		// If all inputs are valid, expect no error
		if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid OS Update Policy creation: %v", err)
		}
	})
}
