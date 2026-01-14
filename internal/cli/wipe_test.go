// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) wipeData(project string, args commandArgs) (string, error) {
	// create application-reference <deployment-package-name> <version> <application-name:version>
	commandString := addCommandArgs(args, fmt.Sprintf(`wipe --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestWipe() {
	// const (
	// 	app1        = "app1"
	// 	app1Version = "1.0"
	// 	pubName     = "pubtest"
	// 	pkgName     = "deployment-pkg"
	// 	pkgVersion  = "1.0"
	// )

	// Test standalone config
	SArgs := map[string]string{
		"yes": "",
	}

	_, err := s.wipeData(project, SArgs)
	s.NoError(err)

	//Missing flag test
	_, err = s.wipeData(project, map[string]string{})
	s.EqualError(err, "you have to say yes")
}
