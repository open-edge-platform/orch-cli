// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"
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
		if name == "" {
			if err == nil {
				t.Errorf("Expected error for missing required field, got: %v", err)
			}
			return
		} else if err != nil && (strings.Contains(err.Error(), "no artifact profile matches the given name") ||
			strings.Contains(err.Error(), "accepts") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "invalid URL escape") ||
			strings.Contains(err.Error(), "invalid control character in URL") ||
			strings.Contains(err.Error(), "required flag \"project\" not set") ||

			strings.Contains(err.Error(), "no such file or directory")) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid network %s creation, type %s: %v", name, ntype, err)
			return
		}

		// --- Set ---
		setArgs := map[string]string{}
		_, err = testSuite.setNetwork("project", name, setArgs)
		if name == "" {
			if err == nil {
				t.Errorf("Expected error for missing network name in set, got: %v", err)
			}
		} else if err != nil && (strings.Contains(err.Error(), "no artifact profile matches the given name") ||
			strings.Contains(err.Error(), "accepts") ||
			strings.Contains(err.Error(), "required flag \"project\" not set") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "invalid URL escape") ||
			strings.Contains(err.Error(), "invalid control character in URL") ||
			strings.Contains(err.Error(), "no such file or directory")) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid network set: %v", err)
		}

		// --- Get ---
		_, err = testSuite.getNetwork("project", name, map[string]string{})
		if name == "" {
			if err == nil {
				t.Errorf("Expected error for missing network name in get, got: %v", err)
			}
		} else if err != nil && (strings.Contains(err.Error(), "no artifact profile matches the given name") ||
			strings.Contains(err.Error(), "accepts") ||
			strings.Contains(err.Error(), "required flag \"project\" not set") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "invalid URL escape") ||
			strings.Contains(err.Error(), "invalid control character in URL") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 0") ||

			strings.Contains(err.Error(), "no such file or directory")) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid network get: %v", err)
		}

		// --- List ---
		_, err = testSuite.listNetwork("project", map[string]string{})
		if err != nil && (strings.Contains(err.Error(), "no artifact profile matches the given name") ||
			strings.Contains(err.Error(), "accepts") ||
			strings.Contains(err.Error(), "required flag \"project\" not set") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "invalid URL escape") ||
			strings.Contains(err.Error(), "invalid control character in URL") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 0") ||

			strings.Contains(err.Error(), "no such file or directory")) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid network list: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteNetwork("project", name, map[string]string{})
		if name == "" {
			if err == nil {
				t.Errorf("Expected error for missing network name in delete, got: %v", err)
			}
		} else if err != nil && (strings.Contains(err.Error(), "no artifact profile matches the given name") ||
			strings.Contains(err.Error(), "accepts") ||
			strings.Contains(err.Error(), "required flag \"project\" not set") ||
			strings.Contains(err.Error(), "unknown shorthand flag:") ||
			strings.Contains(err.Error(), "invalid URL escape") ||
			strings.Contains(err.Error(), "invalid control character in URL") ||
			strings.Contains(err.Error(), "accepts 1 arg(s), received 0") ||

			strings.Contains(err.Error(), "no such file or directory")) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid network delete: %v", err)
		}
	})
}
