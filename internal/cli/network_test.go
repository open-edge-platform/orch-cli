// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
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
