// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createSSHKey(project string, name string, path string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create sshkey %s %s --project %s`, name, path, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listSSHKey(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list sshkey --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getSSHKey(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get sshkey "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteSSHKey(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete sshkey "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestSSHKey() {

	name := "admin"
	resourceID := "localaccount-abc12345"

	/////////////////////////////
	// Test SSH Key Creation
	/////////////////////////////

	//create sshkey
	SArgs := map[string]string{}
	_, err := s.createSSHKey(project, name, "./testdata/testpublickey.pub", SArgs)
	s.NoError(err)

	//create with invalid path
	_, err = s.createSSHKey(project, name, "invalid/path.pub", SArgs)
	s.EqualError(err, "failed to read ssh key file: open invalid/path.pub: no such file or directory")

	//create with invalid key
	_, err = s.createSSHKey(project, name, "./testdata/invalidtestpublickey.pub", SArgs)
	s.EqualError(err, "invalid ssh key format: must be ssh-ed25519 or ecdsa-sha2-nistp521")

	/////////////////////////////
	// Test SSH Key Listing
	/////////////////////////////

	//List SSH keys

	SArgs = map[string]string{}
	listOutput, err := s.listSSHKey(project, SArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Remote User": name,
			"Resource ID": resourceID,
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List SSH Key --verbose
	SArgs = map[string]string{
		"verbose": "true",
	}
	listOutput, err = s.listSSHKey(project, SArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"Remote User": name,
			"Resource ID": resourceID,
			"In use":      "No",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	/////////////////////////////
	// Test SSH Key Get
	/////////////////////////////

	getOutput, err := s.getSSHKey(project, name, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Remote User Name:": name,
		"Resource ID:":      resourceID,
		"Key:":              "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7... admin@example.com",
		"In use by:":        "",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	/////////////////////////////
	// Test SSH Key Delete
	/////////////////////////////

	//delete custom config
	_, err = s.deleteSSHKey(project, name, make(map[string]string))
	s.NoError(err)

	//delete invalid custom config
	_, err = s.deleteSSHKey(project, "nonexistent-key", make(map[string]string))
	s.EqualError(err, "no SSH key matches the given name")

}

func FuzzSSHKey(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "key1", "./testdata/testpublickey.pub")

	f.Fuzz(func(t *testing.T, project, name, path string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{}

		// Call your SSH Key creation logic (replace with your actual function if needed)
		_, err := testSuite.createSSHKey(project, name, path, args)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listSSHKey(project, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get ---
		_, err = testSuite.getSSHKey(project, name, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteSSHKey(project, name, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
