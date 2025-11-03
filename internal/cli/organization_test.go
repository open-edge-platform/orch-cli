// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createOrganization(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create organization  %s --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listOrganization(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list organizations --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getOrganization(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get organization "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteOrganization(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete organization "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestOrganization() {

	name := "itep"
	status := "STATUS_INDICATION_IDLE"
	description := "itep"
	CArgs := map[string]string{}

	/////////////////////////////
	// Test Organization Creation
	/////////////////////////////

	//create organization
	_, err := s.createOrganization(project, name, CArgs)
	s.NoError(err)

	CArgs = map[string]string{
		"description": "test",
	}
	_, err = s.createOrganization(project, name, CArgs)
	s.NoError(err)

	/////////////////////////////
	// Test Organization Listing
	/////////////////////////////

	//List organizations

	listOutput, err := s.listOrganization(project, make(map[string]string))
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name":   name,
			"Status": status,
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List organizations --verbose
	CArgs = map[string]string{
		"verbose": "true",
	}
	listOutput, err = s.listOrganization(project, CArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"Name":        name,
			"Status":      status,
			"Description": description,
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	/////////////////////////////
	// Test Organization Get
	/////////////////////////////

	getOutput, err := s.getOrganization(project, name, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Name:":           name,
		"Description:":    description,
		"Status:":         status,
		"Status message:": "Org itep CREATE is complete",
		"UID:":            "db8d42ad-849d-4626-8dc7-d7955b83e995",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	/////////////////////////////
	// Test Organization Delete
	/////////////////////////////

	//delete organization
	_, err = s.deleteOrganization(project, name, make(map[string]string))
	s.NoError(err)

	//delete invalid organization
	_, err = s.deleteOrganization(project, "nonexistent-org", make(map[string]string))
	s.EqualError(err, "error deleting organization nonexistent-org: Not Found")

}

func FuzzOrganization(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "test", "my desc")
	f.Add("project", "", "my dec")
	f.Add("project", "test", "")

	f.Fuzz(func(t *testing.T, project, name, description string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"description": description,
		}

		_, err := testSuite.createOrganization(project, name, args)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listOrganization(project, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get ---
		_, err = testSuite.getOrganization(project, name, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteOrganization(project, name, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
