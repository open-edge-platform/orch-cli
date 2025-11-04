// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createProject(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create project  %s --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listProject(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list projects --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getProject(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get project "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteProject(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete project "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestProject() {

	name := "itep"
	status := "STATUS_INDICATION_IDLE"
	description := "itep"
	CArgs := map[string]string{}

	/////////////////////////////
	// Test Project Creation
	/////////////////////////////

	//create project
	_, err := s.createProject(project, name, CArgs)
	s.NoError(err)

	CArgs = map[string]string{
		"description": "test",
	}
	_, err = s.createProject(project, name, CArgs)
	s.NoError(err)

	/////////////////////////////
	// Test Project Listing
	/////////////////////////////

	//List projects

	listOutput, err := s.listProject(project, make(map[string]string))
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name":   name,
			"Status": status,
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List projects --verbose
	CArgs = map[string]string{
		"verbose": "true",
	}
	listOutput, err = s.listProject(project, CArgs)
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
	// Test Project Get
	/////////////////////////////

	getOutput, err := s.getProject(project, name, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Name:":           name,
		"Description:":    description,
		"Status:":         status,
		"Status message:": "Project itep CREATE is complete",
		"UID:":            "70883f2f-4bbe-4a67-9eea-1a5824dee549",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	/////////////////////////////
	// Test Project Delete
	/////////////////////////////

	//delete project
	_, err = s.deleteProject(project, name, make(map[string]string))
	s.NoError(err)

	//delete invalid project
	_, err = s.deleteProject(project, "nonexistent-project", make(map[string]string))
	s.EqualError(err, "error deleting project nonexistent-project: Not Found")

}

func FuzzProject(f *testing.F) {
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

		_, err := testSuite.createProject(project, name, args)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listProject(project, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get ---
		_, err = testSuite.getProject(project, name, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteProject(project, name, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
