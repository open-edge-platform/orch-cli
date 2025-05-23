// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) createCluster(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create cluster --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestCluster() {
	_, err := s.createCluster(project, make(map[string]string))
	fmt.Printf("createCluster: %v\n", err)
	s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)
}
