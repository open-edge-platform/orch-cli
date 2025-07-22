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
	_, err := s.listClusterTemplates(project, make(map[string]string))
	s.NoError(err)
}
