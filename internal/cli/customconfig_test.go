// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createCustomConfig(project string, name string, path string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create customconfig %s %s --project %s`, name, path, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listCustomConfig(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list customconfig --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getCustomConfig(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get customconfig "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteCustomConfig(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete customconfig "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestCustomConfig() {

	name := "nginx-config"
	path := "./testdata/cloudinit.yaml"
	resourceID := "config-abc12345"
	timestamp := "2025-01-15 10:30:00 +0000 UTC"
	description := "Nginx configuration for web services"
	CArgs := map[string]string{}

	/////////////////////////////
	// Test CustomConfig Creation
	/////////////////////////////

	//invalid path
	_, err := s.createCustomConfig(project, name, "notest", CArgs)
	s.EqualError(err, "open notest: no such file or directory")

	//invalid name
	_, err = s.createCustomConfig(project, "&*5sd", "notest", CArgs)
	s.EqualError(err, "input is not an alphanumeric single word")

	//creat customconfig
	_, err = s.createCustomConfig(project, name, path, CArgs)
	s.NoError(err)

	CArgs = map[string]string{
		"description": "test",
	}
	_, err = s.createCustomConfig(project, name, path, CArgs)
	s.NoError(err)

	/////////////////////////////
	// Test Custom Config Listing
	/////////////////////////////

	//List customconfig

	listOutput, err := s.listCustomConfig(project, make(map[string]string))
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name":        name,
			"Resource ID": resourceID,
			"Description": description,
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List customconfig --verbose
	CArgs = map[string]string{
		"verbose": "true",
	}
	listOutput, err = s.listCustomConfig(project, CArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"Name":               name,
			"Resource ID":        resourceID,
			"Description":        description,
			"Creation Timestamp": timestamp,
			"Updated Timestamp":  timestamp,
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	/////////////////////////////
	// Test Custom Config Get
	/////////////////////////////

	getOutput, err := s.getCustomConfig(project, name, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Name:":        "nginx-config",
		"Resource ID:": "config-abc12345",
		"Description:": "Nginx configuration for web services",
		"Cloud Init:":  "",
		"test:":        "",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	/////////////////////////////
	// Test Custom Config Delete
	/////////////////////////////

	//delete custom config
	_, err = s.deleteCustomConfig(project, name, make(map[string]string))
	s.NoError(err)

	//delete invalid cusotm config
	_, err = s.deleteCustomConfig(project, "nonexistent-config", make(map[string]string))
	s.EqualError(err, "no custom config matches the given name")

}

func FuzzCustomConfig(f *testing.F) {
	// Seed with valid and invalid input combinations
	f.Add("test-config", "./testdata/cloudinit.yaml", "description", project)
	f.Add("", "./testdata/cloudinit.yaml", "", project)
	f.Add("test-config", "", "", project)
	f.Add("test-config", "blabla", "asfsf", "")

	f.Fuzz(func(t *testing.T, configName, path, description, publisher string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"description": description,
		}

		// --- Create CustomConfig ---
		_, err := testSuite.createCustomConfig(publisher, configName, path, args)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List CustomConfig ---
		_, err = testSuite.listCustomConfig(publisher, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get CustomConfig ---
		_, err = testSuite.getCustomConfig(publisher, configName, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete CustomConfig ---
		_, err = testSuite.deleteCustomConfig(publisher, configName, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
