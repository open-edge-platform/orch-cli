// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) listCharts(project string, registry string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get charts %s --project %s`, registry, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestCharts() {

	registry := "my-registry"
	//TODO only testing fail as not feasible to mock at this time
	_, err := s.listCharts(project, registry, map[string]string{})
	s.Error(err)

}
