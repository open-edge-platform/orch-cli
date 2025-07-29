// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) upload(project string, path string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`upload %s --project %s`, path, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestUpload() {

	// TODO rework upload/test to work with mocks
	_, err := s.upload(project, "./testdata/upload-file.yaml", map[string]string{})
	s.Error(err)

	_, err = s.upload(project, "./testdata/no-upload-file.yaml", map[string]string{})
	s.Error(err)
}
