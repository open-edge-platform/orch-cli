// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createNetwork(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create network %s --project %s`, name, project))
	return s.runCommand(commandString)
}
func (s *CLITestSuite) setNetwork(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`set network %s --project %s`, name, project))
	return s.runCommand(commandString)
}
func (s *CLITestSuite) getNetwork(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get network %s --project %s`, name, project))
	return s.runCommand(commandString)
}
func (s *CLITestSuite) deleteNetwork(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete network %s --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listNetwork(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list networks --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestNetwork() {

	// //TODO revisit these tests
	// Test create network
	_, err := s.createNetwork(project, "my-network", map[string]string{})
	s.NoError(err)

	//Test set network
	_, err = s.setNetwork(project, "my-network", map[string]string{})
	s.NoError(err)

	// Test get network
	_, err = s.getNetwork(project, "my-network", map[string]string{})
	s.NoError(err)

	// Test list networks
	_, err = s.listNetwork(project, map[string]string{})
	s.NoError(err)

	// Test delete network
	_, err = s.deleteNetwork(project, "my-network", map[string]string{})
	s.NoError(err)

}

func FuzzNetwork(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "net1", "private")
	f.Add("project", "", "private")         // missing network name
	f.Add("", "net1", "private")            // missing project
	f.Add("project", "net1", "")            // missing type
	f.Add("project", "net1", "invalidtype") // invalid type

	f.Fuzz(func(t *testing.T, project, name, ntype string) {
		_ = project //todo use properly
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		createArgs := map[string]string{
			"type": ntype,
		}

		// --- Create ---
		_, err := testSuite.createNetwork("project", name, createArgs)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Set ---
		setArgs := map[string]string{}
		_, err = testSuite.setNetwork("project", name, setArgs)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get ---
		_, err = testSuite.getNetwork("project", name, map[string]string{})
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listNetwork("project", map[string]string{})
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteNetwork("project", name, map[string]string{})
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
