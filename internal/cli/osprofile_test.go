// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) createOSProfile(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create osprofile --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listOSProfile(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list osprofile --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getOSProfile(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get osprofile --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteOSProfile(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete osprofile --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestOSProfile() {
	_, err := s.createOSProfile(project, make(map[string]string))
	fmt.Printf("createOSProfile: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.listOSProfile(project, make(map[string]string))
	fmt.Printf("listOSProfile: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.getOSProfile(project, make(map[string]string))
	fmt.Printf("getOSProfile: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.deleteOSProfile(project, make(map[string]string))
	fmt.Printf("deleteOSProfile: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)
}
