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

func FuzzOSUpdatePolicy(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "./testdata/mutableosupdateprofile.yaml", "osupdatepolicy-abc12345")
	f.Add("project", "./testdata/latestosupdateprofile.yaml", "osupdatepolicy-abc12345")
	f.Add("project", "", "osupdatepolicy-abc12345")                                               // missing file
	f.Add("project", "./testdata/invalid.yaml", "osupdatepolicy-abc12345")                        // invalid file
	f.Add("invalid-project", "./testdata/mutableosupdateprofile.yaml", "osupdatepolicy-abc12345") // invalid project
	f.Add("project", "./testdata/duplicate.yaml", "osupdatepolicy-abc12345")                      // duplicate name
	f.Add("", "./testdata/mutableosupdateprofile.yaml", "osupdatepolicy-abc12345")                // missing project
	f.Add("project", "./testdata/mutableosupdateprofile.yaml", "")                                // missing id for delete/list

	f.Fuzz(func(t *testing.T, project, path, id string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{}

		// --- Create ---
		_, err := testSuite.createOSUpdatePolicy(project, path, args)
		if path != "./testdata/mutableosupdateprofile.yaml" && path != "./testdata/latestosupdateprofile.yaml" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing or invalid file, got: %v", err)
			}
		} else if project == "invalid-project" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for invalid project, got: %v", err)
			}
		} else if strings.Contains(path, "duplicate") {
			if err == nil || !strings.Contains(err.Error(), "already exists") {
				t.Errorf("Expected error for duplicate OS Update Policy name, got: %v", err)
			}
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid OS Update Policy creation: %v", err)
		}

		// --- List ---
		_, err = testSuite.listOSUpdatePolicy(project, args)
		if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid OS Update Policy list: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.getOSUpdatePolicy(project, id, args)
		if id == "" || strings.TrimSpace(id) == "" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing id in get, got: %v", err)
			}
		} else if err != nil && (strings.Contains(err.Error(), "Internal Server Error") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 2")) {
			// Acceptable error for backend/server issues
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid OS Update Policy get: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteOSUpdatePolicy(project, id, args)
		if id == "" || strings.TrimSpace(id) == "" {
			if !testSuite.Error(err) {
				t.Errorf("Expected error for missing id in delete, got: %v", err)
			}
		} else if err != nil && (strings.Contains(err.Error(), "Internal Server Error") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 2")) {
			// Acceptable error for backend/server issues
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid OS Update Policy delete: %v", err)
		}
	})
}
