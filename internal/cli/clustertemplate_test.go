// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) listClusterTemplates(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list clustertemplates --project %s`,
		publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestClusterTemplate() {

	/////////////////////////////
	// Test Cluster List
	/////////////////////////////

	//List cluster
	CArgs := map[string]string{}
	_, err := s.listClusterTemplates(project, CArgs)
	s.NoError(err)

	//List cluster --verbose
	CArgs = map[string]string{
		"verbose": "",
	}
	_, err = s.listClusterTemplates(project, CArgs)
	s.NoError(err)
}
