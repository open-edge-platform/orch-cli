// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) version(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`version --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestVersion() {

	_, err := s.version(project, map[string]string{})
	s.NoError(err)
}
