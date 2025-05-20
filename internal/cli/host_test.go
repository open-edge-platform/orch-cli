// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) registerHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`register host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deauthorizeHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`deauthorize host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestHost() {
	_, err := s.registerHost(project, make(map[string]string))
	fmt.Printf("registerHost: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.listHost(project, make(map[string]string))
	fmt.Printf("listHost: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.getHost(project, make(map[string]string))
	fmt.Printf("getHost: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.deauthorizeHost(project, make(map[string]string))
	fmt.Printf("deauthorizeHost: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	_, err = s.deleteHost(project, make(map[string]string))
	fmt.Printf("deleteHost: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)
}
