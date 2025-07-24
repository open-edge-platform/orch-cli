// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) generateStandaloneConfig(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`generate standalone-config --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestStandalone() {
	filename := "./testdata/config-file"

	/////////////////////////////
	// Test Standalone Generation
	/////////////////////////////

	// Save original function and restore after test
	originalFunc := getPasswordFromUser
	defer func() { getPasswordFromUser = originalFunc }()

	// Mock the function to return test password
	getPasswordFromUser = func(_ string) (string, error) {
		return "pass", nil
	}

	// Test standalone config
	SArgs := map[string]string{
		"config-file": filename,
	}

	_, err := s.generateStandaloneConfig(project, SArgs)
	s.NoError(err)

	// Test standalone config with specific output
	SArgs = map[string]string{
		"config-file": filename,
		"output-file": "my-config.yaml",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.NoError(err)

	//Preload with user apps
	SArgs = map[string]string{
		"config-file": filename,
		"user-apps":   "",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.NoError(err)

	//Preload with version
	SArgs = map[string]string{
		"config-file":       filename,
		"emts-repo-version": "a28db5e6d2d9fb6ec5368246c13bfff7fc1a1ae2",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.NoError(err)

	//Invalid config
	SArgs = map[string]string{
		"config-file": "nope",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.EqualError(err, "failed to parse YAML block: open nope: no such file or directory")

	//No config
	SArgs = map[string]string{}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.EqualError(err, "required flag \"config-file\" not set")

	//Preload with wrong repo version
	SArgs = map[string]string{
		"config-file":       filename,
		"emts-repo-version": "sadasdasdsd",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.EqualError(err, "bad status: 404 Not Found")

	// Test standalone config with specific invalid output
	SArgs = map[string]string{
		"config-file": filename,
		"output-file": "/",
	}

	_, err = s.generateStandaloneConfig(project, SArgs)
	s.EqualError(err, "failed to write cloud-init to path \"/\"")

}
