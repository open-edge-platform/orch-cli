// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) watchAll(project string, cmd string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`watch %s --project %s`, cmd, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestWatch() {
	WArgs := map[string]string{}
	//TODO awatch needs refactoring to be more testable with mock
	_, err := s.watchAll(project, "registries", WArgs)
	s.Error(err)

	_, err = s.watchAll(project, "all", WArgs)
	s.Error(err)

}
