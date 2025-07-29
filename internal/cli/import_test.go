// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) importHelmChart(project string, path string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`import helm-chart %s --project %s`, path, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestImport() {
	IArgs := map[string]string{
		"values-file": "./testdata/values.yaml",
	}
	//TODO import needs refactoring to be more testable with mock
	_, err := s.importHelmChart(project, "oci://url", IArgs)
	s.Error(err)

}
